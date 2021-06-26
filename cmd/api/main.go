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
	"github.com/go-chi/cors"
)

func main() {

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"https://*", "http://*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	r.Get("/buyers", GetAllBuyers)
	r.Get("/buyer/{id}/purchase-history", GetPurchaseHistoryByBuyer)
	r.Get("/transaction/date", GetTransactionsOfTheDay)
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
			uid
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

func GetTransactionsOfTheDay(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")

	dg, cancel := getDgraphClient()
	defer cancel()
	txn := dg.NewReadOnlyTxn().BestEffort()
	query :=
		`query all($a: string) {
			transactions_of_the_day(func: eq(date, $a)) {
			  customer {
				name
				age
			  }
			  ip
			  device
			  includes {
				  uid
				  name
				  price
				}
			  date
			  }
		  }`

	ctx := context.Background()
	resp, err := txn.QueryWithVars(ctx, query, map[string]string{"$a": date})

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
