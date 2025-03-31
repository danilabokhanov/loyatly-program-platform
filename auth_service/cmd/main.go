package main

import (
	authhandlers "authservice/auth_handlers"
	pgstorage "authservice/auth_storage/postgresql_storage"
	smimpl "authservice/auth_storage/storage_manager"
	"fmt"
	"log"
	"net/http"
)

func main() {
	r := authhandlers.NewRouter(smimpl.NewStorageManager(pgstorage.NewStorage()))
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatalf("Failed starting server: %v\n", err)
	}
	fmt.Println("Server is running ...")
}
