package main

import (
	authhandlers "authservice/auth_handlers"
	mockstorage "authservice/auth_storage/mock_storage"
	smimpl "authservice/auth_storage/storage_manager"
	"fmt"
	"net/http"
)

func main() {
	r := authhandlers.NewRouter(smimpl.NewStorageManager(mockstorage.NewStorage()))
	fmt.Println("Server is running ...")
	fmt.Printf("%v\n", http.ListenAndServe(":8080", r))
}
