package main

import (
	authhandlers "authservice/auth_handlers"
	pgstorage "authservice/auth_storage/postgresql_storage"
	smimpl "authservice/auth_storage/storage_manager"
	protoauth "authservice/proto/auth"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	server := grpc.NewServer()
	protoauth.RegisterAuthServiceServer(server, authhandlers.NewAuthServer(smimpl.NewStorageManager(pgstorage.NewStorage())))
	reflection.Register(server)

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("Failed to listen:", err)
	}

	log.Println("gRPC server started on :8080")
	if err := server.Serve(listener); err != nil {
		log.Fatal("Failed to serve:", err)
	}
}
