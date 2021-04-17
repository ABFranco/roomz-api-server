package server

import (
  "context"
  "testing"

  rclient "github.com/ABFranco/roomz-api-server/client"
  rpb "github.com/ABFranco/roomz-proto/go_proto"
)

func makeEnterChatRoomReq(roomId, userId uint32, roomToken string) (*rpb.EnterChatRoomRequest) {
  return &rpb.EnterChatRoomRequest{
    RoomId: roomId,
    UserId: userId,
    Token:  roomToken,
  }
}

// TODO: EnterChatRoomHost/Guest/SignedIn
func TestEnterChatRoom(t *testing.T) {
  client := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()

  //
  // test setup
  //
  signInResp := rclient.RandUserSignIn(client)
  if !signInResp.GetSuccess() {
    t.Errorf(":createRoomTest: Rand-sign-in failed")
  }
  _, createRoomResp := rclient.RandRoomCreate(signInResp, false, client)
  if !createRoomResp.GetSuccess() {
    t.Errorf(":createRoomTest: Rand-create-room failed")
  }
  
  userId := signInResp.GetUserId()
  roomId := createRoomResp.GetRoomId()
  roomToken := createRoomResp.GetToken()

  req := makeEnterChatRoomReq(roomId, userId, roomToken)
  _, err := client.EnterChatRoom(context.Background(), req)
  if err != nil {
    t.Errorf(":enterchatroom: failed to enter chatroom")
  }
}