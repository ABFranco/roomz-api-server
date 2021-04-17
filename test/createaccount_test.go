package server

import (
  "context"
  "testing"

  rclient "github.com/ABFranco/roomz-api-server/client"
  "github.com/ABFranco/roomz-api-server/util"
  rpb "github.com/ABFranco/roomz-proto/go_proto"
)

func makeCreateAccountReq(fname, lname, email, password string) *rpb.CreateAccountRequest {
  return &rpb.CreateAccountRequest{
    FirstName: fname,
    LastName:  lname,
    Email:     email,
    Password:  password,
  }
}

func TestCreateAccount(t *testing.T) {
  fname := "xm"
  lname := "quentin"
  email := "ABFranco@gmail.com"
  password := "1234"
  tests := []struct {
    testName string
    req *rpb.CreateAccountRequest
    wantSuccess bool
  }{{
    "validAccountCreationSuccess",
    makeCreateAccountReq(fname, lname, email, password),
    true,
  }, {
    "duplicateAccountFail",
    makeCreateAccountReq(fname, lname, email, password),
    false,
  }, {
    "emptyEmailFail",
    makeCreateAccountReq(fname, lname, "", password),
    false,
  }, {
    "emptyPasswordFail",
    makeCreateAccountReq(fname, lname, email + util.RandStr(), ""),
    false,
  },
  }

  client := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()

  for _, tt := range tests {
    t.Run(tt.testName, func(t *testing.T) {			
      resp, ec := client.CreateAccount(context.Background(), tt.req)
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