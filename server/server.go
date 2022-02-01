package server

import (
	"fmt"

	"github.com/Antonite/oware_rl/storage"
)

type Server struct {
	store *storage.Storage
}

func New() *Server {
	store, err := storage.Init(50)
	if err != nil {
		fmt.Println("failed to initialize storage")
		panic(err)
	}

	return &Server{store: store}
}

func (s *Server) Close() {
	s.store.Close()
}
