package server

import (
  "context"
  "fmt"
  "sync"
  "testing"

  rclient "github.com/ABFranco/roomz-api-server/client"
  rpb "github.com/ABFranco/roomz-proto/go_proto"
)

func makeJoinRoomReq(roomId, userId uint32, roomPassword, userName string, isGuest bool) (*rpb.JoinRoomRequest) {
  return &rpb.JoinRoomRequest{
    RoomId:       roomId,
    RoomPassword: roomPassword,
    UserId:       userId,
    UserName:     userName,
    IsGuest:      isGuest,
  }
}

// TODO: create more tests to verify chat history is successfully sent

func TestJoinNonStrictRoom(t *testing.T) {
  hostClient := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()

  //
  // test setup
  //
  signInResp := rclient.RandUserSignIn(hostClient)
  if !signInResp.GetSuccess() {
    t.Errorf(":createRoomTest: Rand-sign-in failed")
  }
  createRoomReq, createRoomResp := rclient.RandRoomCreate(signInResp, false, hostClient)
  if !createRoomResp.GetSuccess() {
    t.Errorf(":createRoomTest: Rand-create-room failed")
  }

  // create guest user to join the room
  guestClient := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()

  roomIdToJoin := createRoomResp.GetRoomId()
  roomPassword := createRoomReq.GetPassword()
  guestUserName := "bobo"

  req := makeJoinRoomReq(roomIdToJoin, 0, roomPassword, guestUserName, true)
  guestJoinStream, err := guestClient.JoinRoom(context.Background(), req)
  if err != nil {
    t.Errorf(":joinRoomTest: Failed to join non-strict room.")
  }

  // create signed-in user to join the room
  signInClient := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()

  signInResp2 := rclient.RandUserSignIn(signInClient)
  if !signInResp2.GetSuccess() {
    t.Errorf(":createRoomTest: Rand-sign-in failed")
  }
  signInUserName := "jojo"
  req = makeJoinRoomReq(roomIdToJoin, signInResp2.GetUserId(), roomPassword, signInUserName, false)
  signInJoinStream, err2 := signInClient.JoinRoom(context.Background(), req)
  if err2 != nil {
    t.Errorf(":joinRoomTest: Failed to join non-strict room.")
  }

  // await JoinRoomResponse notification
  done := make(chan int)
  var wg sync.WaitGroup

  joinStreams := []rpb.RoomzApiService_JoinRoomClient{guestJoinStream, signInJoinStream}
  for _, stream := range joinStreams {
    wg.Add(1)
    go func(str rpb.RoomzApiService_JoinRoomClient) {
      defer wg.Done()
      for {
        joinResp, err := str.Recv()
        if err != nil {
          fmt.Errorf("Error reading message: %v", err)
          break
        }
        joinStatus := joinResp.GetStatus()
        if joinStatus == "accept" {
          break
        } else if joinStatus == "wait" {
          fmt.Printf("should be waiting!")
          break
        } else if joinStatus == "rejected" {
          fmt.Printf("rejected")
          break
        } else {
          t.Errorf("Incorrect JoinRoom status")
        }
      }
    }(stream)
  }

  go func() {
    wg.Wait()
    close(done)
  }()
  <-done
}

func TestJoinStrictRoom(t *testing.T) {
  hostClient := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()

  //
  // test setup
  //
  signInResp := rclient.RandUserSignIn(hostClient)
  if !signInResp.GetSuccess() {
    t.Errorf(":createRoomTest: Rand-sign-in failed")
  }
  createRoomReq, createRoomResp := rclient.RandRoomCreate(signInResp, true, hostClient)
  if !createRoomResp.GetSuccess() {
    t.Errorf(":createRoomTest: Rand-create-room failed")
  }

  // create guest user to join the room
  guestClient := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()

  roomIdToJoin := createRoomResp.GetRoomId()
  roomPassword := createRoomReq.GetPassword()
  guestUserName := "bobo"

  req := makeJoinRoomReq(roomIdToJoin, 0, roomPassword, guestUserName, true)
  guestJoinStream, err := guestClient.JoinRoom(context.Background(), req)
  if err != nil {
    t.Errorf(":joinRoomTest: Failed to join strict room.")
  }

  // create signed-in user to join the room
  signInClient := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()

  signInResp2 := rclient.RandUserSignIn(signInClient)
  if !signInResp2.GetSuccess() {
    t.Errorf(":createRoomTest: Rand-sign-in failed")
  }
  signInUserName := "jojo"
  req = makeJoinRoomReq(roomIdToJoin, signInResp2.GetUserId(), roomPassword, signInUserName, false)
  signInJoinStream, err2 := signInClient.JoinRoom(context.Background(), req)
  if err2 != nil {
    t.Errorf(":joinRoomTest: Failed to join strict room.")
  }

  // await JoinRoomResponse notification
  done := make(chan int)
  var wg sync.WaitGroup

  joinStreams := []rpb.RoomzApiService_JoinRoomClient{guestJoinStream, signInJoinStream}
  for _, stream := range joinStreams {
    wg.Add(1)
    go func(str rpb.RoomzApiService_JoinRoomClient) {
      defer wg.Done()
      for {
        joinResp, err := str.Recv()
        if err != nil {
          fmt.Errorf("Error reading message: %v", err)
          break
        }
        joinStatus := joinResp.GetStatus()
        if joinStatus == "accept" {
          break
        } else if joinStatus == "wait" {
          fmt.Printf("should be waiting!\n")
          break
        } else if joinStatus == "rejected" {
          fmt.Printf("rejected\n")
          break
        } else {
          t.Errorf("Incorrect JoinRoom status")
        }
      }
    }(stream)
  }

  go func() {
    wg.Wait()
    close(done)
  }()
  <-done
}