package server

import (
  "context"
  "testing"
  "time"

  rclient "github.com/ABFranco/roomz-api-server/client"
  rpb "github.com/ABFranco/roomz-proto/go_proto"
)

func makeGetJoinRequestsReq(room_id, user_id uint32) *rpb.GetJoinRequestsRequest {
  return &rpb.GetJoinRequestsRequest{
    RoomId: room_id,
    UserId: user_id,
  }
}

func TestGetJoinRequestsTest(t *testing.T) {
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
  
  // create guest user to join the room
  guestClient := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()

  roomPassword := createRoomReq.GetPassword()
  guestUserName := "bobo"

  // user sents join request, gets blocked
  var err error
  joinRoomReq := makeJoinRoomReq(roomId, 0, roomPassword, guestUserName, true)
  _, err = guestClient.JoinRoom(context.Background(), joinRoomReq)
  if err != nil {
    t.Errorf(":joinRoomTest: Failed to join non-strict room.")
  }

  // As long as completion_time(JoinRoom) is before completion_time(GetJoinRequests),
  // we're fine...so try not to be too expensive.
  time.Sleep(50 * time.Millisecond)

  // host now can query join requests
  req := makeGetJoinRequestsReq(roomId, hostUserId)
  var resp *rpb.GetJoinRequestsResponse
  resp, err = hostClient.GetJoinRequests(context.Background(), req)
  if err != nil {
    t.Errorf(":getJoinRequests: Failed to get join requests.")
  }

  if !resp.GetSuccess() {
    t.Errorf(":getJoinRequests: response is \"unsuccessful\"")
  }
  if resp.GetRoomId() != roomId {
    t.Errorf(":getJoinRequests: roomId in response does not match")
  }
  var strictJoinRequests []*rpb.StrictJoinRequest
  strictJoinRequests = resp.GetJoinRequests()

  if len(strictJoinRequests) != 1 {
    t.Errorf(":getJoinRequests: received more StrictJoinRequests than expected")
  }
  for _, strictJoinRequest := range strictJoinRequests {
    if strictJoinRequest.GetRoomId() != roomId {
      t.Errorf(":getJoinRequests: roomId in room join request is not correct")
    }
    // TODO: get guest JoinRoomResponse to capture user_id. But would need to receive off of stream
    if strictJoinRequest.GetUserName() != guestUserName {
      t.Errorf(":getJoinRequests: user_name not correct in room join request")
    }
  }
}

func TestGetMultipleJoinRequestsTest(t *testing.T) {
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
  
  // create guest user to join the room
  guestClient1 := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()

  guestUserName1 := "bobo"
  // user sents join request, gets blocked
  var err error
  joinRoomReq1 := makeJoinRoomReq(roomId, 0, roomPassword, guestUserName1, true)
  _, err = guestClient1.JoinRoom(context.Background(), joinRoomReq1)
  if err != nil {
    t.Errorf(":joinRoomTest: Failed to join non-strict room.")
  }

  // create another guest user to join the room
  guestClient2 := rclient.NewRoomzApiServiceClient()
  defer rclient.CloseRoomzApiServiceClient()

  guestUserName2 := "dodo"
  // user sents join request, gets blocked
  joinRoomReq2 := makeJoinRoomReq(roomId, 0, roomPassword, guestUserName2, true)
  _, err = guestClient2.JoinRoom(context.Background(), joinRoomReq2)
  if err != nil {
    t.Errorf(":joinRoomTest: Failed to join non-strict room.")
  }

  // As long as completion_time(JoinRoom) is before completion_time(GetJoinRequests),
  // we're fine...so try not to be too expensive.
  time.Sleep(50 * time.Millisecond)

  // host now can query join requests
  req := makeGetJoinRequestsReq(roomId, hostUserId)
  var resp *rpb.GetJoinRequestsResponse
  resp, err = hostClient.GetJoinRequests(context.Background(), req)
  if err != nil {
    t.Errorf(":getJoinRequests: Failed to get join requests.")
  }

  if !resp.GetSuccess() {
    t.Errorf(":getJoinRequests: response is \"unsuccessful\"")
  }
  if resp.GetRoomId() != roomId {
    t.Errorf(":getJoinRequests: roomId in response does not match")
  }
  var strictJoinRequests []*rpb.StrictJoinRequest
  strictJoinRequests = resp.GetJoinRequests()

  if len(strictJoinRequests) != 2 {
    t.Errorf(":getJoinRequests: received more StrictJoinRequests than expected")
  }
  for _, strictJoinRequest := range strictJoinRequests {
    if strictJoinRequest.GetRoomId() != roomId {
      t.Errorf(":getJoinRequests: roomId in room join request is not correct")
    }
    // TODO: get guest JoinRoomResponse to capture user_id. But would need to receive off of stream
    if strictJoinRequest.GetUserName() != guestUserName1 && strictJoinRequest.GetUserName() != guestUserName2  {
      t.Errorf(":getJoinRequests: user_name not correct in room join request")
    }
  }
}