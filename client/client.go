package client

import (
	"log"

	rpb "github.com/ABFranco/roomz-proto/go_proto"
	"google.golang.org/grpc"
)

var (
	cc *grpc.ClientConn
)

func NewRoomzApiServiceClient() rpb.RoomzApiServiceClient {
	var err error
	opts := grpc.WithInsecure()
	cc, err = grpc.Dial("localhost:9090", opts)
	if err != nil {
		log.Fatal(err)
	}
	return rpb.NewRoomzApiServiceClient(cc)
}

func CloseRoomzApiServiceClient() {
	cc.Close()
}