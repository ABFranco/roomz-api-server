package server

import (
  "context"
  "testing"

  rclient "github.com/ABFranco/roomz-api-server/client"
  rpb "github.com/ABFranco/roomz-proto/go_proto"
)

func makeSignInReq(email, password string) *rpb.SignInRequest {
  return &rpb.SignInRequest{
    Email: email,
    Password: password,
  }
}

func TestSignIn(t *testing.T) {
  client := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()

  //
  // test setup
  //
  createAccountReq, createAccountResp := rclient.CreateRandomAccount(client)
  if !createAccountResp.Success {
    t.Errorf(":signInTest: Could not create valid account")
  }
  actEmail := createAccountReq.GetEmail()
  actPassword := createAccountReq.GetPassword()

  tests := []struct {
    testName string
    req *rpb.SignInRequest
    wantSuccess bool
  }{ {
    "stdSignInSuccess",
    makeSignInReq(actEmail, actPassword),
    true,
  }, {
    "incorrectPasswordFail",
    makeSignInReq(actEmail, "4321"),
    false,
  }, {
    "incorrectEmailFail",
    makeSignInReq("x@q", actPassword),
    false,
  }, {
    "emptyPasswordFail",
    makeSignInReq(actEmail, ""),
    false,
  }, {
    "emptyEmailFail",
    makeSignInReq("", actPassword),
    false,
  },
  }

  for _, tt := range tests {
    t.Run(tt.testName, func(t *testing.T) {
      resp, ec := client.SignIn(context.Background(), tt.req)
      if ec != nil {
        if tt.wantSuccess {
          t.Errorf("Received error=%v when not expected.", ec.Error())
        }
      } else if resp.Success != tt.wantSuccess {
        t.Errorf("got %v, want %v", resp.Success, tt.wantSuccess)
      }
    })
  }
}