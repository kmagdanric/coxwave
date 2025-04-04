package main

import (
	"log"
	"net"
	"net/http"

	"coupons/internal/server"
)

func main() {
	srv, err := server.New()
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}
	defer srv.Close()

	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("Coupon issuance service is running on :8080")

	httpServer := &http.Server{
		Handler: srv.Handler(),
	}

	if err := httpServer.Serve(lis); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
