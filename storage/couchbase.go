package storage

import (
	"errors"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
)

const (
	// Not concerned about exposing these for this use case
	user    = "oware"
	pass    = "owarerl"
	bucket  = "qlearn"
	workers = 1000
)

type Storage struct {
	collections map[string]*gocb.Collection
	RewardChan  chan string
	PunishChan  chan string
}

type OwareState struct {
	Reward   int
	Children []string
}

func Init() (*Storage, error) {
	cluster, err := gocb.Connect(
		"localhost",
		gocb.ClusterOptions{
			Username: user,
			Password: pass,
		})
	if err != nil {
		return nil, err
	}

	bucket := cluster.Bucket(bucket)
	err = bucket.WaitUntilReady(5*time.Second, nil)
	if err != nil {
		return nil, err
	}

	collections := make(map[string]*gocb.Collection, 2)

	// Player 0 collection
	sc0 := bucket.Scope("0")
	collection0 := sc0.Collection("0")
	collections["0"] = collection0

	// Player 1 collection
	sc1 := bucket.Scope("1")
	collection1 := sc1.Collection("1")
	collections["1"] = collection1

	rewardChan := make(chan string)
	punishChan := make(chan string)
	s := &Storage{
		collections: collections,
		RewardChan:  rewardChan,
		PunishChan:  punishChan,
	}

	// Initialize workers
	s.processRewards()

	return s, nil
}

func (s *Storage) Close() {
	close(s.PunishChan)
	close(s.RewardChan)
	fmt.Println("Closed storage channels")
}

func (s *Storage) Get(key string) (*OwareState, error) {
	r, err := s.collections[key[2:3]].Get(key, nil)
	for err == gocb.ErrDocumentLocked {
		fmt.Printf("document locked: %s\n", key)
		time.Sleep(time.Second)
		r, err = s.collections[key[2:3]].Get(key, nil)
	}
	if err != nil {
		return nil, err
	}

	var state OwareState
	if err := r.Content(&state); err != nil {
		fmt.Printf("failed to parse state. %v\n", err)
		return nil, err
	}

	return &state, nil
}

func (s *Storage) GetAndLock(key string) (*OwareState, gocb.Cas, error) {
	r, err := s.retryGetAndLock(key, time.Second*15, time.Second)
	cas := r.Cas()
	if cas == 0 {
		panic(err)
	}

	var state OwareState
	if err := r.Content(&state); err != nil {
		fmt.Printf("failed to parse state. %v\n", err)
		return nil, cas, err
	}

	return &state, cas, nil
}

func (s *Storage) SafeAddChildren(key string, children []string) error {
	state, cas, err := s.GetAndLock(key)
	defer s.unlock(key, cas)
	if err != nil {
		return err
	}

	if len(state.Children) > 0 {
		return nil
	}

	state.Children = children
	return s.Replace(key, cas, state)
}

func (s *Storage) SafeAdjustReward(key string, adjustment int) error {
	state, cas, err := s.GetAndLock(key)
	defer s.unlock(key, cas)
	if err != nil {
		fmt.Printf("failing to save award. key: %s, cas: %v\n", key, cas)
		return err
	}

	state.Reward += adjustment
	return s.Replace(key, cas, state)
}

func (s *Storage) Replace(key string, cas gocb.Cas, state *OwareState) error {
	_, err := s.collections[key[2:3]].Replace(key, state, &gocb.ReplaceOptions{Cas: cas})
	return err
}

func (s *Storage) Update(key string, state *OwareState) error {
	_, err := s.collections[key[2:3]].Upsert(key, state, nil)
	return err
}

func (s *Storage) processRewards() {
	for w := 1; w <= workers/2; w++ {
		go s.adjust(w, 1, s.RewardChan)
	}

	for w := 1; w <= workers/2; w++ {
		go s.adjust(w, -1, s.PunishChan)
	}
}

func (s *Storage) adjust(id int, reward int, moves <-chan string) {
	for m := range moves {
		if err := s.SafeAdjustReward(m, reward); err != nil {
			fmt.Printf("failed to save reward: %s\n", m)
			panic(err)
		}
	}
}

func (s *Storage) unlock(key string, cas gocb.Cas) {
	if cas == 0 {
		return
	}

	c, exists := s.collections[key[2:3]]
	if !exists {
		fmt.Printf("FAILED TO UNLOCK. %s, %v", key, cas)
		panic("collection doesn't exist")
	}

	c.Unlock(key, cas, nil)
}

func (s *Storage) retryGetAndLock(key string, timeout time.Duration, backoff time.Duration) (*gocb.GetResult, error) {
	c, exists := s.collections[key[2:3]]
	if !exists {
		fmt.Printf("FAILED TO GET AND LOCK. %s", key)
		panic("collection doesn't exist")
	}

	retry := true
	for retry {
		r, err := c.GetAndLock(key, timeout, nil)
		if err == nil {
			return r, nil
		}

		switch t := err.(type) {
		case *gocb.TimeoutError:
			fmt.Printf("get and lock timeout %s\n", key)
			time.Sleep(backoff)
		default:
			fmt.Printf("fail to get and lock - type of error\n: %v", t)
			return nil, err
		}
	}

	return nil, errors.New("failed to retry and lock. loop exited")
}
