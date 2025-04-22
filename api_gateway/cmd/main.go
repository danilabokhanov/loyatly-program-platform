package main

import (
	"apigateway/proxy"
	"fmt"
	"log"
	"net/http"
)

func main() {
	g, err := proxy.NewGrpcClients()
	if err != nil {
		log.Fatalf("Failed create grpc clients: %v\n", err)
	}
	r := proxy.NewRouter(g)
	err = http.ListenAndServe(":8082", r)
	if err != nil {
		log.Fatalf("Failed starting server: %v\n", err)
	}
	fmt.Println("Server is running ...")
}
