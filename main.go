package main

import (
	"log"
	"net/http"

	"github.com/JakeNeyer/ipam/server"
	"github.com/JakeNeyer/ipam/store"
)

func main() {
	st := store.NewStore()
	s := server.NewServer(st)

	log.Println("http://localhost:8011/docs")
	log.Println("On first run, open the app to complete setup and create the initial admin.")
	if err := http.ListenAndServe("localhost:8011", s); err != nil {
		log.Fatal(err)
	}
}
