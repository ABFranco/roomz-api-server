// A part of the client utility package, this file offers other utility
// functions like performing events on randomized objects (such as a unique
// log-in, etc.).
//
// Each exported function is dependent upon adequate unit tests of gRPC
// functionality.
package client

import (
  "context"

  "github.com/ABFranco/roomz-api-server/util"
	rpb "github.com/ABFranco/roomz-proto/go_proto"
)


// CreateRandomAccount creates a random Account object, and returns a
// CreateAccountRequest and CreateAccountResponse object.
func CreateRandomAccount(client rpb.RoomzApiServiceClient) (*rpb.CreateAccountRequest, *rpb.CreateAccountResponse) {
	req := &rpb.CreateAccountRequest{
    FirstName: "xm",
    LastName:  "quentin",
    Email:     "xq" + util.RandStr() + "@gmail.com",
    Password:  "1234",
  }
  resp, _ := client.CreateAccount(context.Background(), req)
  return req, resp
}


// RandUserSignIn creates a random Account object and signs into the newly
// created Account. It returns a SignInResponse object.
func RandUserSignIn(client rpb.RoomzApiServiceClient) (*rpb.SignInResponse) {
  createAccountReq, _ := CreateRandomAccount(client)
  req := &rpb.SignInRequest{
    Email: createAccountReq.GetEmail(),
    Password: createAccountReq.GetPassword(),
  }
  resp, _ := client.SignIn(context.Background(), req)
  return resp
}

// RandRoomCreate creates a new Room and returns a CreateRoomRequest obeject
// and CreateRoomResponse object.
func RandRoomCreate(signInResp *rpb.SignInResponse, isStrict bool, client rpb.RoomzApiServiceClient) (*rpb.CreateRoomRequest, *rpb.CreateRoomResponse) {
  req := &rpb.CreateRoomRequest{
    UserId:   signInResp.GetUserId(),
    UserName: "joe" + util.RandStr(),
    Password: "4321",
    IsStrict: isStrict,
  }
  resp, _ := client.CreateRoom(context.Background(), req)
  return req, resp
}