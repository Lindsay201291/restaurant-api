package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dgraph-io/dgo/v2/protos/api"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func main() {

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/buyers", GetAllBuyers)
	r.Get("/buyer/{id}/purchase-history", GetPurchaseHistoryByBuyer)
	r.Post("/transaction", CreateTransaction)

	http.ListenAndServe(":3000", r)
}

// Buyer
type buyer struct {
	Uid  string `json:"uid,omitempty"`
	Name string `json:"name,omitempty"`
	Age  int    `json:"age,omitempty"`
}

// Product
type Product struct {
	Uid   string  `json:"uid,omitempty"`
	Name  string  `json:"name,omitempty"`
	Price float64 `json:"price,omitempty"`
}

// Transaction
type Transaction struct {
	Uid      string    `json:"uid,omitempty"`
	Buyer    buyer     `json:"buyer,omitempty"`
	Ip       string    `json:"ip,omitempty"`
	Device   string    `json:"device,omitempty"`
	Products []Product `json:"products,omitempty"`
	Date     int64     `json:"date,omitempty"`
}

func GetAllBuyers(w http.ResponseWriter, r *http.Request) {
	// w.Write([]byte("Buyers list"))

	dg, cancel := getDgraphClient()
	defer cancel()
	txn := dg.NewReadOnlyTxn().BestEffort()
	resp, err := txn.Query(context.Background(),
		`{  buyers(func: has(age)) {
			name
			age
		} }`)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(resp.Json))

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	w.Write(resp.Json)
}

func GetPurchaseHistoryByBuyer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	dg, cancel := getDgraphClient()
	defer cancel()
	txn := dg.NewReadOnlyTxn().BestEffort()
	query :=
		`query all($a: string) {  
			purchase_history(func: uid($a)) {
				name
				age
				purchases : ~customer {
				ip
				device
				includes {
					name
					price
				}
				date
				}
			} 
		}`

	ctx := context.Background()
	resp, err := txn.QueryWithVars(ctx, query, map[string]string{"$a": id})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(resp.Json))

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	w.Write(resp.Json)
}

func CreateTransaction(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var transaction Transaction
	err := decoder.Decode(&transaction)
	if err != nil {
		fmt.Fprintf(w, "error: %v", err)
		return
	}

	fmt.Println(transaction.Device)
	fmt.Println(transaction.Products)
	transaction.Date = (time.Now().UnixNano() / 1e6)

	ctx := context.TODO()
	dg, cancel := getDgraphClient()
	defer cancel()

	txn := dg.NewTxn()
	defer txn.Commit(ctx)

	/* t := Transaction{
		Uid: "1624147200",
		Buyer: buyer{
			Name: "Huba",
			Age:  45,
		},
		Ip:     "32.66.100.154",
		Device: "linux",
		Products: []Product{{
			Name:  "Chicken broth",
			Price: 5095,
		}, {
			Name:  "Original ready rice",
			Price: 3511,
		}},
		Date: (time.Now().UnixNano() / 1e6),
	} */

	lb, err := json.Marshal(transaction)
	if err != nil {
		log.Fatal("failed to marshal ", err)
	}

	mu := &api.Mutation{
		SetJson: lb,
	}
	res, err := txn.Mutate(ctx, mu)
	if err != nil {
		log.Fatal("failed to mutate ", err)
	}

	print("res: %v", res)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-type", "application/json")

	w.Write(lb)
}
