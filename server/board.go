package server

import (
	"encoding/json"
	"net/http"

	"github.com/Antonite/oware"
)

type BoardResponse struct {
	Status oware.GameStatus
	Player int
	Scores []int
	Pits   []int
}

func (s *Server) GetBoardHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,HEAD,OPTIONS,POST,PUT")
	w.Header().Set("Access-Control-Allow-Headers", "Access-Control-Allow-Headers, Origin,Accept, X-Requested-With, Content-Type, Access-Control-Request-Method, Access-Control-Request-Headers")

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id is a required param", http.StatusBadRequest)
		return
	}

	board, err := s.getBoard(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(board)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (s *Server) getBoard(id string) (*BoardResponse, error) {
	b, err := oware.NewS(id)
	if err != nil {
		return nil, err
	}

	bresponse := &BoardResponse{
		Status: b.Status,
		Player: b.Player(),
		Scores: b.Scores(),
		Pits:   b.Pits(),
	}

	return bresponse, nil
}
