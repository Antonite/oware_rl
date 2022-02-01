package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/Antonite/oware"
	"github.com/Antonite/oware_rl/agents/tableagent"
)

type MovesResponse struct {
	Id     string
	Pit    int
	Reward int
}

func (s *Server) GetMovesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,HEAD,OPTIONS,POST,PUT")
	w.Header().Set("Access-Control-Allow-Headers", "Access-Control-Allow-Headers, Origin,Accept, X-Requested-With, Content-Type, Access-Control-Request-Method, Access-Control-Request-Headers")

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id is a required param", http.StatusBadRequest)
		return
	}

	moves, err := s.getMoves(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(moves)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (s *Server) getMoves(id string) ([]*MovesResponse, error) {
	mresponse := []*MovesResponse{}

	a := tableagent.New(s.store)
	b, err := oware.NewS(id)
	if err != nil {
		return mresponse, err
	}

	a.SetBoard(b)
	moves := a.Board().GetValidMoves()

	// Get possible moves with reward values
	moveMap := a.ExploreCurrentMoves(moves, id)
	for _, m := range moves {
		nb, err := a.Board().Move(m)
		if err != nil {
			continue
		}

		nbs := nb.ToString()

		reward, ok := moveMap[nbs]
		if !ok {
			fmt.Printf("couldn't find move in storage: %s, for key: %s\n", nbs, id)
			return mresponse, errors.New("couldn't find move in storage")
		}

		mresponse = append(mresponse, &MovesResponse{
			Id:     nbs,
			Pit:    m,
			Reward: reward,
		})
	}

	return mresponse, nil
}
