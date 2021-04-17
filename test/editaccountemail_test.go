package server

import (
  "context"
  "testing"

  rpb "github.com/ABFranco/roomz-proto/go_proto"
  rclient "github.com/ABFranco/roomz-api-server/client"
  "google.golang.org/grpc/codes"
  "google.golang.org/grpc/status"
)

func makeEditAccountEmailReq(oldEmail, newEmail string) *rpb.EditAccountEmailRequest {
  return &rpb.EditAccountEmailRequest{
    OldEmail: oldEmail,
    NewEmail: newEmail,
  }
}

func TestEditAccountEmail(t *testing.T) {
  client := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()
  createAccountReq, createAccountResp := rclient.CreateRandomAccount(client)
  if !createAccountResp.Success {
    t.Errorf(":signInTest: Could not create valid account")
  }
  actEmail := createAccountReq.GetEmail()
  newEmail := "ABFranco@gmail.com"
  tests := []struct {
    testName string
    req *rpb.EditAccountEmailRequest
    wantErrorCode codes.Code
  }{{
    "validEmailChangeSuccess",
    makeEditAccountEmailReq(actEmail, newEmail),
    codes.OK,
  }, {
    "nonexistentEmailFail", //email doesn't exist in the database
    makeEditAccountEmailReq("x@q", newEmail),
    codes.Unauthenticated,
  }, {
    "emptyOldEmailFail",
    makeEditAccountEmailReq("", newEmail),
    codes.Unauthenticated,
  }, {
    "emptyNewEmailFail",
    makeEditAccountEmailReq(actEmail, ""),
    codes.Unauthenticated,
  },
  }

  for _, tt := range tests {
    t.Run(tt.testName, func(t *testing.T) {			
      _, gotError := client.EditAccountEmail(context.Background(), tt.req)
      if status.Code(gotError) != tt.wantErrorCode {
        t.Errorf("Received error=%v when not expected.", gotError.Error())
      }
    })
  }
}