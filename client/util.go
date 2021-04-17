// this file performs events on randomized objects (such as a unique log-in, etc.)
//
// each exported function is dependent upon adequate unit tests of gRPC functionality
package client

import (
  "context"

  "github.com/ABFranco/roomz-api-server/util"
	rpb "github.com/ABFranco/roomz-proto/go_proto"
)

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

func RandUserSignIn(client rpb.RoomzApiServiceClient) (*rpb.SignInResponse) {
  createAccountReq, _ := CreateRandomAccount(client)
  req := &rpb.SignInRequest{
    Email: createAccountReq.GetEmail(),
    Password: createAccountReq.GetPassword(),
  }
  resp, _ := client.SignIn(context.Background(), req)
  return resp
}

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