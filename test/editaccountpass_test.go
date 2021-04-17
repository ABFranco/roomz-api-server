package server

import (
  "context"
  "testing"

  rpb "github.com/ABFranco/roomz-proto/go_proto"
  rclient "github.com/ABFranco/roomz-api-server/client"
  "google.golang.org/grpc/codes"
  "google.golang.org/grpc/status"
)

func makeEditAccountPasswordReq(email, oldPass, newPass string) *rpb.EditAccountPasswordRequest {
  return &rpb.EditAccountPasswordRequest{
    Email: email,
    OldPassword: oldPass,
    NewPassword: newPass,
  }
}


func TestEditAccountPassword(t *testing.T) {
  client := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()
  createAccountReq, createAccountResp := rclient.CreateRandomAccount(client)
  if !createAccountResp.Success {
    t.Errorf(":signInTest: Could not create valid account")
  }
  actEmail := createAccountReq.GetEmail()
  actPass := createAccountReq.GetPassword()
  newPass := "1223rjwelj4a"
  tests := []struct {
    testName      string
    req          *rpb.EditAccountPasswordRequest
    wantErrorCode codes.Code
  }{{
    "validPasswordChangeSuccess",
    makeEditAccountPasswordReq(actEmail, actPass, newPass),
    codes.OK,
  }, {
    "nonexistentEmailFail",
    makeEditAccountPasswordReq("p#m", actPass, newPass),
    codes.Unauthenticated,
  }, {
    "incorrectPasswordFail",
    makeEditAccountPasswordReq(actEmail, "s", newPass),
    codes.Unauthenticated,
  }, {
    "emptyEmailFail",
    makeEditAccountPasswordReq("", actPass, newPass),
    codes.Unauthenticated,
  }, {
    "emptyOldPasswordFail",
    makeEditAccountPasswordReq(actEmail, "", newPass),
    codes.Unauthenticated,
  }, {
    "emptyNewPasswordFail",
    makeEditAccountPasswordReq(actEmail, actPass, ""),
    codes.Unauthenticated,
  },
  } 

  for _, tt := range tests {
    t.Run(tt.testName, func(t *testing.T) {			
      _, gotError := client.EditAccountPassword(context.Background(), tt.req)
      if status.Code(gotError) != tt.wantErrorCode {
        t.Errorf("Received error=%v when not expected.", gotError.Error())
      } 
    })
  }
}