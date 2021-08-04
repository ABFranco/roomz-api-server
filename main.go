package main

import (
  "log"
  "net"

  "github.com/ABFranco/roomz-api-server/server"
)

func main() {
  const listenAddress = "0.0.0.0:9090"
  lis, err := net.Listen("tcp", listenAddress)
  if err != nil {
    log.Fatalf("Failed to listen: %v", err)
  }

  server := server.NewRoomzApiServer()
  if server != nil {
    log.Printf("Roomz API Service starting on %v", listenAddress)
    if err := server.Serve(lis); err != nil {
      log.Fatalf("Failed to serve: %v", err)
    }
  }
}