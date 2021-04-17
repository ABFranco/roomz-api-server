package server

import (
  "context"
  "fmt"
  "sync"
  "testing"
  "time"

  rclient "github.com/ABFranco/roomz-api-server/client"
  rpb "github.com/ABFranco/roomz-proto/go_proto"
)

func makeCloseRoomReq(roomId uint32) (*rpb.CloseRoomRequest) {
  return &rpb.CloseRoomRequest{
    RoomId: roomId,
  }
}

func makeAwaitRoomClosureReq(roomId, userId uint32, roomToken string) (*rpb.AwaitRoomClosureRequest) {
  return &rpb.AwaitRoomClosureRequest{
    RoomId: roomId,
    UserId: userId,
    Token:  roomToken,
  }
}

func TestCloseRoom(t *testing.T) {
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

  var closeStream rpb.RoomzApiService_AwaitRoomClosureClient
  var err error
  awaitRoomClosureReq := makeAwaitRoomClosureReq(roomId, userId, roomToken)
  closeStream, err = client.AwaitRoomClosure(context.Background(), awaitRoomClosureReq)
  if err != nil {
    t.Errorf(":awaitroomclosure: failed to await room closure")
  }
  time.Sleep(5*time.Millisecond)

  // await HostClosedRoom notification
  done := make(chan int)
  var wg sync.WaitGroup
  wg.Add(1)
  go func(str rpb.RoomzApiService_AwaitRoomClosureClient) {
    defer wg.Done()
    for {
      _, err := str.Recv()
      if err != nil {
        fmt.Errorf("Error reading message: %v", err)
        break
      }
      fmt.Printf("Received room closure notification. Shutting down...\n")
      break
    }
  }(closeStream)

  // send close room request
  req := makeCloseRoomReq(roomId)
  resp, err := client.CloseRoom(context.Background(), req)
  if err != nil {
    t.Errorf("Failed to send CloseRoomRequest")
  }
  if !resp.GetSuccess() {
    t.Errorf("Failed to close room")
  }
  fmt.Printf("Successfully closed room.\n")

  // wait for Done()
  go func() {
    wg.Wait()
    close(done)
  }()
  <-done
}