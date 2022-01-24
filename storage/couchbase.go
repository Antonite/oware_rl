package storage

import (
	"errors"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
)

const (
	// Not concerned about exposing these for this use case
	user   = "oware"
	pass   = "owarerl"
	bucket = "qlearn"
)

type Storage struct {
	collections map[string]*gocb.Collection
	RewardChan  chan string
	PunishChan  chan string
}

type OwareState struct {
	Reward   int
	Children []string
	Games    int
}

func Init(workers int) (*Storage, error) {
	cluster, err := gocb.Connect(
		"localhost",
		gocb.ClusterOptions{
			Username:             user,
			Password:             pass,
			CircuitBreakerConfig: gocb.CircuitBreakerConfig{Disabled: true},
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
	s.processRewards(workers)

	return s, nil
}

func (s *Storage) Close() {
	close(s.PunishChan)
	close(s.RewardChan)
	fmt.Println("Closed storage channels")
}

func (s *Storage) Get(key string) (*OwareState, error) {
	c, exists := s.collections[key[2:3]]
	if !exists {
		return nil, errors.New("collection doesn't exist")
	}

	retry := true
	retries := 1
	var r *gocb.GetResult
	var err error
	for retry {
		r, err = c.Get(key, nil)
		if err == nil {
			retry = false
			continue
		}

		switch err.(type) {
		case *gocb.KeyValueError:
			return nil, err
		default:
		}

		retries++
		time.Sleep(time.Millisecond * 100 * time.Duration(retries))
		if retries > 30 {
			fmt.Printf("get error #%v key %s\n", retries, key)
		}
	}

	var state OwareState
	if err := r.Content(&state); err != nil {
		fmt.Printf("failed to parse state. %v\n", err)
		return nil, err
	}

	return &state, nil
}

func (s *Storage) GetAndLock(key string) (*OwareState, gocb.Cas, error) {
	r, _ := s.retryGetAndLock(key, time.Second*15)
	cas := r.Cas()

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
	state.Games++
	return s.Replace(key, cas, state)
}

func (s *Storage) Replace(key string, cas gocb.Cas, state *OwareState) error {
	c, exists := s.collections[key[2:3]]
	if !exists {
		return errors.New("collection doesn't exist")
	}

	retry := true
	retries := 1
	for retry {
		_, err := c.Replace(key, state, &gocb.ReplaceOptions{Cas: cas})
		if err == nil {
			return nil
		}

		retries++
		time.Sleep(time.Millisecond * 100 * time.Duration(retries))
		if retries > 20 {
			return err
		}
	}

	return errors.New("failed to replace. loop exited")
}

func (s *Storage) Insert(key string, state *OwareState) error {
	c, exists := s.collections[key[2:3]]
	if !exists {
		return errors.New("collection doesn't exist")
	}

	retry := true
	retries := 1
	for retry {
		_, err := c.Insert(key, state, nil)
		if err == nil {
			return nil
		}

		retries++
		time.Sleep(time.Millisecond * 100 * time.Duration(retries))
		if retries > 20 {
			switch t := err.(type) {
			default:
				fmt.Printf("insert error #%v key %s type %v\n", retries, key, t)
			}
			return err
		}
	}

	return errors.New("failed to update. loop exited")
}

func (s *Storage) processRewards(workers int) {
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
	}

	c.Unlock(key, cas, nil)
}

func (s *Storage) retryGetAndLock(key string, timeout time.Duration) (*gocb.GetResult, error) {
	c, exists := s.collections[key[2:3]]
	if !exists {
		return nil, errors.New("collection doesn't exist")
	}

	retry := true
	retries := 1
	for retry {
		r, err := c.GetAndLock(key, timeout, nil)
		if err == nil {
			return r, nil
		}

		retries++
		time.Sleep(time.Millisecond * 100 * time.Duration(retries))
		if retries > 20 {
			fmt.Printf("get and lock error #%v key %s\n", retries, key)
		}
	}

	return nil, errors.New("failed to retry and lock. loop exited")
}
