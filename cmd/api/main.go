package main

import (
	"io"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func main() {

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/buyers", getAllBuyers)

	http.ListenAndServe(":3000", r)
}

type Link struct {
	Uid   string   `json:"uid,omitempty"`
	URL   string   `json:"url,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`
}

func getAllBuyers(w http.ResponseWriter, r *http.Request) {
	{
		// w.Write([]byte("Buyers list"))
		resp, err := http.Get("https://kqxty15mpg.execute-api.us-east-1.amazonaws.com/buyers")

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		defer resp.Body.Close()

		w.Header().Set("Content-Type", "application/json")

		if _, err := io.Copy(w, resp.Body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
