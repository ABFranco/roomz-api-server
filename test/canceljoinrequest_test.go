package server

import (
  "context"
  "sync"
  "testing"

  rclient "github.com/ABFranco/roomz-api-server/client"
  rpb "github.com/ABFranco/roomz-proto/go_proto"
)

func makeCancelJoinRequestReq(room_id, user_id uint32) *rpb.CancelJoinRequestRequest {
  return &rpb.CancelJoinRequestRequest{
    RoomId: room_id,
    UserId: user_id,
  }
}

func TestCancelJoinRequest(t *testing.T) {
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

  // have to await response so we can capture user ID
  var guestUserId uint32

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

      guestUserId = joinResp.GetUserId()
      wg.Done()
      break
    }
  }(guestJoinStream)
  go func() {
    wg.Wait()
    close(done)
  }()
  <-done

  // host should verify we have room join requests
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

  // now have the guest cancel their join request
  req := makeCancelJoinRequestReq(roomId, guestUserId)
  resp, err3 := guestClient1.CancelJoinRequest(context.Background(), req)
  if err3 != nil {
    t.Errorf(":cancelJoinRequest: CancelJoinRequest RPC failed")
  }
  if !resp.GetSuccess() {
    t.Errorf(":cancelJoinRequest: CancelJoinRequest response was unsuccessful")
  }

  // now have the host again query the getJoinRequests, and verify it at zero.
  joinRequests, err = hostClient.GetJoinRequests(context.Background(), getJoinRequestsReq)
  if err != nil {
    t.Errorf(":cancelJoinRequest: Host failed to getJoinRequests")
  }
  if len(joinRequests.GetJoinRequests()) > 0 {
    t.Errorf(":cancelJoinRequest: len(joinRequests) should be zero after a cancellation.")
  }
}