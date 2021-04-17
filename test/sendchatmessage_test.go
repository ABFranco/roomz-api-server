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

func makeChatMessage(roomId, userId uint32, userName, message, timestamp string) (*rpb.ChatMessage) {
  return &rpb.ChatMessage{
    RoomId: roomId,
    UserId: userId,
    UserName: userName,
    Message: message,
    Timestamp: timestamp,
  }
}

func TestSendChatMessage(t *testing.T) {
  client := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()

  //
  // test setup
  //
  signInResp := rclient.RandUserSignIn(client)
  if !signInResp.GetSuccess() {
    t.Errorf(":createRoomTest: Rand-sign-in failed")
  }
  createRoomReq, createRoomResp := rclient.RandRoomCreate(signInResp, false, client)
  if !createRoomResp.GetSuccess() {
    t.Errorf(":createRoomTest: Rand-create-room failed")
  }
  userId := signInResp.GetUserId()
  userName := createRoomReq.GetUserName()
  roomId := createRoomResp.GetRoomId()
  roomToken := createRoomResp.GetToken()

  enterChatRoomReq := &rpb.EnterChatRoomRequest{
    RoomId: roomId,
    UserId: userId,
    Token:  roomToken,
  }
  chatStream, err := client.EnterChatRoom(context.Background(), enterChatRoomReq)
  if err != nil {
    t.Errorf(":enterchatroom: failed to enter chatroom")
  }


  // await chat messages
  var wg sync.WaitGroup
  done := make(chan int)
  wg.Add(1)
  go func(str rpb.RoomzApiService_EnterChatRoomClient) {
    defer wg.Done()
    for {
      msg, err := str.Recv()
      if err != nil {
        fmt.Errorf("Error reading message: %v", err)
        break
      }
      fmt.Printf("Received=[%v] %v: %s\n", msg.GetTimestamp(), msg.GetUserName(), msg.GetMessage())
      break
    }
  }(chatStream)

  // emit a chat message
  wg.Add(1)
  go func() {
    defer wg.Done()

    msg := makeChatMessage(roomId, userId, userName, "hello world", fmt.Sprintf("%v", time.Now()))
    _, err := client.SendChatMessage(context.Background(), msg)
    if err != nil {
      t.Errorf("Failed to send chat message")
    }
    fmt.Printf("Sent chat message!\n")
  }()

  // await both goroutines' Done()
  go func() {
    wg.Wait()
    close(done)
  }()
  <-done
}