package server

import (
  "context"
  "testing"

  rclient "github.com/ABFranco/roomz-api-server/client"
  rpb "github.com/ABFranco/roomz-proto/go_proto"
)

func makeCreateRoomReq(userId uint32, userName, roomPassword string, isStrict bool) (*rpb.CreateRoomRequest) {
  return &rpb.CreateRoomRequest{
    UserId:   userId,
    UserName: userName,
    Password: roomPassword,
    IsStrict: isStrict,
  }
}

func TestCreateRoom(t *testing.T) {
  client := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()
  //
  // test setup
  //
  signInResp := rclient.RandUserSignIn(client)
  if !signInResp.GetSuccess() {
    t.Errorf(":createRoomTest: Rand-sign-in failed")
  }

  userName := "joe"
  roomPassword := "4321"
  userId := signInResp.GetUserId()

  tests := []struct {
    testName string
    req *rpb.CreateRoomRequest
    wantSuccess bool
  }{ {
    "CreateNonStrictRoomSuccess",
    makeCreateRoomReq(userId, userName, roomPassword, false),
    true,
  }, {
    "CreateStrictRoomSuccess",
    makeCreateRoomReq(userId, userName, roomPassword, true),
    true,
  },
  }

  for _, tt := range tests {
    t.Run(tt.testName, func(t *testing.T) {
      resp, ec := client.CreateRoom(context.Background(), tt.req)
      if ec != nil {
        if tt.wantSuccess {
          t.Errorf("Received unexpected error=%v", ec.Error())
        }
      } else if resp.Success != tt.wantSuccess {
        t.Errorf("got %v, want %v", resp.Success, tt.wantSuccess)
      }
    })
  }
}