package main

import (
	"context"
	"log"
	"net/http"

	"github.com/dgraph-io/dgo/v2/protos/api"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
)

func main() {

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Use(cors.Handler(cors.Options{

		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	r.Get("/buyers", GetAllBuyers)
	r.Get("/buyer/{id}/purchase-history", GetPurchaseHistoryByBuyer)
	r.Get("/buyer/{id}/same-ip", GetOtherBuyersWithTheSameIp)
	r.Get("/buyer/{id}/product-recomendations", GetProductRecomendations)
	r.Get("/buyer/date", GetBuyersOfTheDay)
	r.Get("/product/date", GetProductsOfTheDay)
	r.Get("/transaction/date", GetTransactionsOfTheDay)

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

func RunQueryWithVars(query string, variable string) *api.Response {

	dg, cancel := getDgraphClient()
	defer cancel()
	txn := dg.NewReadOnlyTxn().BestEffort()

	ctx := context.Background()
	resp, err := txn.QueryWithVars(ctx, query, map[string]string{"$a": variable})

	if err != nil {
		log.Fatal(err)
	}

	return resp
}

func RunQuery(query string) *api.Response {

	dg, cancel := getDgraphClient()
	defer cancel()
	txn := dg.NewReadOnlyTxn().BestEffort()

	resp, err := txn.Query(context.Background(),
		query)

	if err != nil {
		log.Fatal(err)
	}

	return resp
}

func GetAllBuyers(w http.ResponseWriter, r *http.Request) {

	query := `{  buyers(func: has(age)) {
		uid
		name
		age
	} }`

	resp := RunQuery(query)

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	w.Write(resp.Json)
}

func GetBuyersOfTheDay(w http.ResponseWriter, r *http.Request) {

	date := r.URL.Query().Get("date")
	query :=
		`query all($a: string) {
			var(func: eq(date, $a)) {
				customer {
					customers as uid
				}
			}
			q(func: uid(customers)) {
				name
				age
			}
		}`

	resp := RunQueryWithVars(query, date)

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	w.Write(resp.Json)
}

func GetPurchaseHistoryByBuyer(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
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

	resp := RunQueryWithVars(query, id)

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	w.Write(resp.Json)
}

func GetOtherBuyersWithTheSameIp(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
	query :=
		`query all($a: string) {
			var(func: uid($a)) {
				~customer {
					ips as ip
				}
			}  
			q(func: eq(ip, val(ips))) {
				customer @filter (NOT uid($a)) {
					uid
					name
					c : ~customer @filter (eq(ip, val(ips))) {
						ip
					}
				}
			}
		}`

	resp := RunQueryWithVars(query, id)

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	w.Write(resp.Json)
}

func GetProductRecomendations(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
	query :=
		`query all($a: string) {
			var(func: uid($a)) {
				~customer {
					includes {
						products as uid
					}
				}
			}
			q(func: has(ip), first: 5) @cascade {
				ip
				customer {
				   name
				   age
				}
				includes @filter(uid(products)) {
				   name
				   price
				 }
			}
		}`

	resp := RunQueryWithVars(query, id)

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	w.Write(resp.Json)
}

func GetProductsOfTheDay(w http.ResponseWriter, r *http.Request) {

	date := r.URL.Query().Get("date")
	query :=
		`query all($a: string) {
			var(func: eq(date, $a)) {
				includes {
					products as uid
				}
			}
			q(func: uid(products)) {
				name
				price
			}
		}`

	resp := RunQueryWithVars(query, date)

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	w.Write(resp.Json)
}

func GetTransactionsOfTheDay(w http.ResponseWriter, r *http.Request) {

	date := r.URL.Query().Get("date")
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

	resp := RunQueryWithVars(query, date)

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	w.Write(resp.Json)
}
