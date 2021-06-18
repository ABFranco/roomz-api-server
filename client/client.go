// The client package is a utility package for testing using a similated gRPC
// client.
package client

import (
	"log"

	rpb "github.com/ABFranco/roomz-proto/go_proto"
	"google.golang.org/grpc"
)

var (
	cc *grpc.ClientConn
)

// NewRoomzApiServiceClient returns a new gRPC RoomzApiServiceClient.
func NewRoomzApiServiceClient() rpb.RoomzApiServiceClient {
	var err error
	opts := grpc.WithInsecure()
	cc, err = grpc.Dial("localhost:9090", opts)
	if err != nil {
		log.Fatal(err)
	}
	return rpb.NewRoomzApiServiceClient(cc)
}


// CloseRoomzApiServiceClient closes the gRPC client connection for a
// RoomzApiServiceClient.
func CloseRoomzApiServiceClient() {
	cc.Close()
}