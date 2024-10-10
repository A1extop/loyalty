package main

import (
	"log"
	"net/http"

	"github.com/A1extop/loyalty/config"
	http2 "github.com/A1extop/loyalty/internal/http"
	psql "github.com/A1extop/loyalty/internal/store"
)

func main() {

	parameters := config.NewParameters()
	parameters.Get()
	db, err := psql.ConnectDB(parameters.AddrDB)
	if err != nil {
		log.Println("Failed to connect to database at startup:", err)
	}
	store := psql.NewStore(db)
	repos := http2.NewRepository(store)
	if db != nil {
		psql.CreateOrConnectTable(db)
	}
	router := http2.NewRouter(repos)

	log.Printf("Starting server on port %s", parameters.AddressHTTP)
	err = http.ListenAndServe(parameters.AddressHTTP, router)
	if err != nil {
		log.Fatal(err)
	}

}
