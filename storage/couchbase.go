package storage

import (
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
)

type Storage struct {
	collections map[string]*gocb.Collection
}

type OwareState struct {
	Reward   int
	Children []string
}

const (
	// Not concerned about exposing these for this use case
	user   = "oware"
	pass   = "owarerl"
	bucket = "qlearn"
)

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

	return &Storage{collections: collections}, nil
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

func (s *Storage) GetAndLock(key string) (*OwareState, *gocb.Cas, error) {
	r, err := s.collections[key[2:3]].GetAndLock(key, time.Second*15, nil)
	for err == gocb.ErrDocumentLocked {
		fmt.Printf("document locked: %s\n", key)
		time.Sleep(time.Second)
		r, err = s.collections[key[2:3]].GetAndLock(key, time.Second*15, nil)
	}
	if err != nil {
		return nil, nil, err
	}

	cas := r.Cas()

	var state OwareState
	if err := r.Content(&state); err != nil {
		fmt.Printf("failed to parse state. %v\n", err)
		return nil, &cas, err
	}

	return &state, &cas, nil
}

func (s *Storage) SafeAddChildren(key string, children []string) error {
	state, cas, err := s.GetAndLock(key)
	defer s.collections[key[2:3]].Unlock(key, *cas, nil)
	if err != nil {
		return err
	}

	if len(state.Children) > 0 {
		return nil
	}

	state.Children = children
	return s.Replace(key, *cas, state)
}

func (s *Storage) SafeAdjustReward(key string, adjustment int) error {
	state, cas, err := s.GetAndLock(key)
	defer s.collections[key[2:3]].Unlock(key, *cas, nil)
	if err != nil {
		fmt.Printf("failing to save award. key: %s, cas: %v\n", key, *cas)
		return err
	}

	state.Reward += adjustment
	return s.Replace(key, *cas, state)
}

func (s *Storage) Replace(key string, cas gocb.Cas, state *OwareState) error {
	_, err := s.collections[key[2:3]].Replace(key, state, &gocb.ReplaceOptions{Cas: cas})
	return err
}

func (s *Storage) Update(key string, state *OwareState) error {
	_, err := s.collections[key[2:3]].Upsert(key, state, nil)
	return err
}
