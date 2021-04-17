package server

import (
  "context"
  "sync"
  "testing"
  "time"

  rclient "github.com/ABFranco/roomz-api-server/client"
  rpb "github.com/ABFranco/roomz-proto/go_proto"
)

func makeHandleJoinRequestReq(room_id, user_id uint32, decision string) *rpb.HandleJoinRequestRequest {
  return &rpb.HandleJoinRequestRequest{
    RoomId: room_id,
    UserIdToHandle: user_id,
    Decision: decision,
  }
}

func TestHandleJoinRequestAccept(t *testing.T) {
  hostClient := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()

  //
  // test setup
  //
  signInResp := rclient.RandUserSignIn(hostClient)
  if !signInResp.GetSuccess() {
    t.Errorf(":createRoomTest: Rand-sign-in failed")
  }
  // create a strict-room
  createRoomReq, createRoomResp := rclient.RandRoomCreate(signInResp, true, hostClient)
  if !createRoomResp.GetSuccess() {
    t.Errorf(":createRoomTest: Rand-create-room failed")
  }
  
  hostUserId := signInResp.GetUserId()
  roomId := createRoomResp.GetRoomId()
  roomPassword := createRoomReq.GetPassword()


  // have guest try to enter, query a response
  // create guest user to join the room
  guestClient1 := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()

  guestUserName1 := "bobo"
  // user sents join request, gets blocked
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
  wg.Add(1)
  go func(str rpb.RoomzApiService_JoinRoomClient) {
    numResponses := 0
    for {
      joinResp, err := str.Recv()
      if err != nil {
        t.Errorf("Error reading message: %v", err)
        break
      }
      if numResponses == 0 && joinResp.GetStatus() != "wait" {
        t.Errorf(":handleJoinRequest: guest should receive a wait before a decision")
      }
      if numResponses > 0 && joinResp.GetStatus() != "accept" {
        t.Errorf(":handleJoinRequest: guest did not receive acceptance! Received=%v", joinResp)
      }
      numResponses += 1
      if numResponses == 2 {
        wg.Done()
        break
      }
    }
  }(guestJoinStream)

  // have the host accept the user
  getJoinRequestsReq := makeGetJoinRequestsReq(roomId, hostUserId)
  var joinRequests *rpb.GetJoinRequestsResponse
  joinRequests, err = hostClient.GetJoinRequests(context.Background(), getJoinRequestsReq)
  if err != nil {
    t.Errorf(":handleJoinRequestsTest: Failed to get join requests")
  }
  if len(joinRequests.GetJoinRequests()) == 0 {
    t.Errorf(":handleJoinRequestsTest: No join requests!")
  }
  selectedJoinRequest := joinRequests.GetJoinRequests()[0]
  req := makeHandleJoinRequestReq(roomId, selectedJoinRequest.GetUserId(), "accept")
  var resp *rpb.HandleJoinRequestResponse
  resp, err = hostClient.HandleJoinRequest(context.Background(), req)
  if err != nil {
    t.Errorf(":handleJoinRequestsTest: Failed to handle join request!")
  }
  if !resp.GetSuccess() {
    t.Errorf(":handleJoinRequestsTest: Handle Join Request unsuccessful")
  }

  go func() {
    wg.Wait()
    close(done)
  }()
  <-done
}

func TestHandleJoinRequestReject(t *testing.T) {
  hostClient := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()

  //
  // test setup
  //
  signInResp := rclient.RandUserSignIn(hostClient)
  if !signInResp.GetSuccess() {
    t.Errorf(":createRoomTest: Rand-sign-in failed")
  }
  // create a strict-room
  createRoomReq, createRoomResp := rclient.RandRoomCreate(signInResp, true, hostClient)
  if !createRoomResp.GetSuccess() {
    t.Errorf(":createRoomTest: Rand-create-room failed")
  }
  
  hostUserId := signInResp.GetUserId()
  roomId := createRoomResp.GetRoomId()
  roomPassword := createRoomReq.GetPassword()


  // have guest try to enter, query a response
  // create guest user to join the room
  guestClient1 := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()

  guestUserName1 := "bobo"
  // user sents join request, gets blocked
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
  wg.Add(1)
  go func(str rpb.RoomzApiService_JoinRoomClient) {
    numResponses := 0
    for {
      joinResp, err := str.Recv()
      if err != nil {
        t.Errorf("Error reading message: %v", err)
        break
      }
      if numResponses == 0 && joinResp.GetStatus() != "wait" {
        t.Errorf(":handleJoinRequest: guest should receive a wait before a decision")
      }
      if numResponses > 0 && joinResp.GetStatus() != "reject" {
        t.Errorf(":handleJoinRequest: guest did not receive rejection! Received=%v", joinResp)
      }
      numResponses += 1
      if numResponses == 2 {
        wg.Done()
        break
      }
    }
  }(guestJoinStream)

  // have the host accept the user
  getJoinRequestsReq := makeGetJoinRequestsReq(roomId, hostUserId)
  var joinRequests *rpb.GetJoinRequestsResponse
  joinRequests, err = hostClient.GetJoinRequests(context.Background(), getJoinRequestsReq)
  if err != nil {
    t.Errorf(":handleJoinRequestsTest: Failed to get join requests")
  }
  if len(joinRequests.GetJoinRequests()) == 0 {
    t.Errorf(":handleJoinRequestsTest: No join requests!")
  }
  selectedJoinRequest := joinRequests.GetJoinRequests()[0]
  req := makeHandleJoinRequestReq(roomId, selectedJoinRequest.GetUserId(), "reject")
  var resp *rpb.HandleJoinRequestResponse
  resp, err = hostClient.HandleJoinRequest(context.Background(), req)
  if err != nil {
    t.Errorf(":handleJoinRequestsTest: Failed to handle join request!")
  }
  if !resp.GetSuccess() {
    t.Errorf(":handleJoinRequestsTest: Handle Join Request unsuccessful")
  }

  go func() {
    wg.Wait()
    close(done)
  }()
  <-done
}