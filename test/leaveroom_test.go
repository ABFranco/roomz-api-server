package server

import (
  "context"
  "sync"
  "testing"
  "time"

  rclient "github.com/ABFranco/roomz-api-server/client"
  rpb "github.com/ABFranco/roomz-proto/go_proto"
)

func makeLeaveRoomReq(room_id, user_id uint32) *rpb.LeaveRoomRequest {
  return &rpb.LeaveRoomRequest{
    RoomId: room_id,
    UserId: user_id,
  }
}

func TestLeaveRoom(t *testing.T) {
  hostClient := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()

  //
  // test setup
  //
  signInResp := rclient.RandUserSignIn(hostClient)
  if !signInResp.GetSuccess() {
    t.Errorf(":createRoomTest: Rand-sign-in failed")
  }
  // create a non-strict-room
  createRoomReq, createRoomResp := rclient.RandRoomCreate(signInResp, false, hostClient)
  if !createRoomResp.GetSuccess() {
    t.Errorf(":createRoomTest: Rand-create-room failed")
  }
  
  roomId := createRoomResp.GetRoomId()
  roomPassword := createRoomReq.GetPassword()


  // have guest try to enter, query a response
  // create guest user to join the room
  guestClient1 := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()

  guestUserName1 := "bobo"
  var err error
  joinRoomReq1 := makeJoinRoomReq(roomId, 0, roomPassword, guestUserName1, true)
  guestJoinStream, err2 := guestClient1.JoinRoom(context.Background(), joinRoomReq1)
  if err2 != nil {
    t.Errorf(":joinRoomTest: Failed to join non-strict room.")
  }
  time.Sleep(50 * time.Millisecond)

  // spawn off guest join stream to wait for acceptance
  // await JoinRoomResponse notification
  done := make(chan int)
  var wg sync.WaitGroup
  joinRoomResp := &rpb.JoinRoomResponse{}
  wg.Add(1)
  go func(str rpb.RoomzApiService_JoinRoomClient) {
    for {
      var err error
      joinRoomResp, err = str.Recv()
      if err != nil {
        t.Errorf("Error reading message: %v", err)
        break
    }
    if joinRoomResp.GetStatus() == "accept" {
      wg.Done()
      break
    }
    }
  }(guestJoinStream)

  go func() {
    wg.Wait()
    close(done)
  }()
  <-done

  // since the user is in the room, we can open up some streams
  guestUserId := joinRoomResp.GetUserId()
  guestToken := joinRoomResp.GetToken()
  enterChatroomReq := makeEnterChatRoomReq(roomId, guestUserId, guestToken)
  _, err = guestClient1.EnterChatRoom(context.Background(), enterChatroomReq)
  if err != nil {
    t.Errorf(":enterchatroom: failed to enter chatroom")
  }
  // await close room
  awaitRoomClosureReq := makeAwaitRoomClosureReq(roomId, guestUserId, guestToken)
  _, err = guestClient1.AwaitRoomClosure(context.Background(), awaitRoomClosureReq)
  if err != nil {
    t.Errorf(":awaitroomclosure: failed to await room closure")
  }

  // now leave the room
  req := makeLeaveRoomReq(roomId, guestUserId)
  var resp *rpb.LeaveRoomResponse
  resp, err = guestClient1.LeaveRoom(context.Background(), req)
  if err != nil {
    t.Errorf(":leaveRoomTest: error occurred")
  }
  if !resp.Success {
    t.Errorf(":leaveRoomTest: failed to leave room")
  }
}