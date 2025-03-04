package main

import (
	"apigateway/proxy"
	"fmt"
	"log"
	"net/http"
)

func main() {
	r := proxy.NewRouter()
	err := http.ListenAndServe(":8082", r)
	if err != nil {
		log.Fatalf("Failed starting server: %v\n", err)
	}
	fmt.Println("Server is running ...")
}
