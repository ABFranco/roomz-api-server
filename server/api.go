// The file hosts all Roomz API logic, a part of the server package.
package server

import (
  "fmt"
  "context"
  "log"
  "math/rand"
  "strconv"
  "sync"
  "time"

  "github.com/ABFranco/roomz-api-server/models"
  "github.com/ABFranco/roomz-api-server/util"
  rpb "github.com/ABFranco/roomz-proto/go_proto"
  "google.golang.org/grpc/codes"
  "google.golang.org/grpc/status"
  "gorm.io/gorm"
)

func rollbackTx(tx *gorm.DB) {
  if tx != nil {
    tx.Rollback()
  }
}

func commitTx(tx *gorm.DB) {
  if tx != nil {
    tx.Commit()
  }
}


// CreateAccount creates a new User and Account given an email and password.
func (r *roomzApiService) CreateAccount(ctx context.Context, req *rpb.CreateAccountRequest) (*rpb.CreateAccountResponse, error) {
  log.Printf(":CreateAccount: Received data=%v", req)
  resp := &rpb.CreateAccountResponse{}
  tx := r.RDB.Begin()

  if len(req.Email) == 0 {
    rollbackTx(tx)
    return nil, status.Error(codes.Unauthenticated, "Invalid email, needs to be nonempty.")
  }
  if len(req.Password) == 0 {
    rollbackTx(tx)
    return nil, status.Error(codes.Unauthenticated, "Invalid password, needs to be nonempty.")
  }

  account, res := r.RDB.GetAccount(tx, req.Email)
  if res != gorm.ErrRecordNotFound {
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, "An account already exists with this email!")
  }

  var user models.User
  fullName := fmt.Sprintf("%s %s", req.FirstName, req.LastName)
  user.Name.String = fullName
  if err := r.RDB.CreateUser(tx, &user); err != nil {
    log.Printf(":CreateAccount: Failed to create User. Err=%v", err)
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, "Internal server error.")
  }

  account = models.Account{
    UserId:    user.Id,
    Email:     req.Email,
    Password:  util.Encrypt(req.Password),
    FirstName: req.FirstName,
    LastName:  req.LastName,
    StartDate: time.Now(),
  }
  if err := r.RDB.CreateAccount(tx, &account); err != nil {
    log.Printf(":CreateAccount: Failed to create Account. Err=%v", err)
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, "Internal server error.")
  }
  resp.Success = true
  // TODO: Make user.Id uint32, this can cause overflow if negative.
  resp.UserId = uint32(user.Id)
  commitTx(tx)
  log.Printf(":CreateAccount: resp=%v", resp)
  return resp, nil
}


// SignIn verifies email and password matches a valid Account, and returns all
// User info including userId.
func (r *roomzApiService) SignIn(ctx context.Context, req *rpb.SignInRequest) (*rpb.SignInResponse, error) {
  log.Printf(":SignIn: Received data=%v", req)
  resp := &rpb.SignInResponse{}
  tx := r.RDB.Begin()

  if len(req.Email) == 0 {
    rollbackTx(tx)
    return nil, status.Error(codes.Unauthenticated, "Invalid email, must be nonempty.")
  }
  if len(req.Password) == 0 {
    rollbackTx(tx);
    return nil, status.Error(codes.Unauthenticated, "Invalid password, must be nonempty.")
  }

  var account models.Account
  var err error
  if account, err = r.RDB.GetAccount(tx, req.Email);  err == gorm.ErrRecordNotFound {
    rollbackTx(tx)
    return nil, status.Error(codes.Unauthenticated, "An account does not exist with this email.")
  }
  if util.Decrypt(account.Password) != req.Password {
    rollbackTx(tx)
    return nil, status.Error(codes.Unauthenticated, "Incorrect password, try again!")
  }

  resp = &rpb.SignInResponse{
    Success: true,
    UserId: uint32(account.Id),
    FirstName: account.FirstName,
    LastName: account.LastName,
    ErrorMessage: "",
  }
  log.Printf(":SignIn: resp=%v", resp)
  return resp, nil
}


// EditAccountEmail verifies an old email matches a valid account, and if so,
// replaces the old email with the new email.
// TODO: Add additional security here and in EditAccountPassword in the future.
func (r *roomzApiService) EditAccountEmail(ctx context.Context, req *rpb.EditAccountEmailRequest) (*rpb.EditAccountEmailResponse, error) {
  log.Printf(":EditAccountEmail: Received data=%v", req)
  tx := r.RDB.Begin()
  if len(req.OldEmail) == 0 {
    rollbackTx(tx)
    return nil, status.Error(codes.Unauthenticated, "Current email cannot be empty.")
  }
  if len(req.NewEmail) == 0 {
    rollbackTx(tx)
    return nil, status.Error(codes.Unauthenticated, "New email must be nonempty.")
  }
  // check if "new" email is already in use
  _, res := r.RDB.GetAccount(tx, req.NewEmail)
  if res != gorm.ErrRecordNotFound {
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, "New email invalid. An account already exists with this email!")
  }
  err := r.RDB.EditAccountEmail(tx, req.OldEmail, req.NewEmail)
  if err == gorm.ErrRecordNotFound {
    rollbackTx(tx)
    return nil, status.Error(codes.Unauthenticated, "An account does not exist with this email.")
  }
  resp := &rpb.EditAccountEmailResponse{}
  log.Printf(":EditAccountEmail: resp=%v", resp)
  return resp, nil
}


// EditAccountPassword verifies an email and old password matches a valid account, and if so,
// replaces the old password with the new password.
func (r *roomzApiService) EditAccountPassword(ctx context.Context, req *rpb.EditAccountPasswordRequest) (*rpb.EditAccountPasswordResponse, error) {
  log.Printf(":EditAccountPassword: Received data=%v", req)
  tx := r.RDB.Begin()

  if len(req.Email) == 0 {
    rollbackTx(tx)
    return nil, status.Error(codes.Unauthenticated, "Invalid email, must be nonempty.")
  }
  if len(req.OldPassword) == 0 {
    rollbackTx(tx)
    return nil, status.Error(codes.Unauthenticated, "Invalid password, must be nonempty.")
  }

  account, err := r.RDB.GetAccount(tx, req.Email)
  if err == gorm.ErrRecordNotFound {
    rollbackTx(tx)
    return nil, status.Error(codes.Unauthenticated, "An account does not exist with this email.")
  }
  if util.Decrypt(account.Password) != req.OldPassword {
    rollbackTx(tx)
    return nil, status.Error(codes.Unauthenticated, "Incorrect Password, try again!")
  }
  if len(req.NewPassword) == 0 {
    rollbackTx(tx)
    return nil, status.Error(codes.Unauthenticated, "New password must be nonempty.")
  }

  err = r.RDB.EditAccountPassword(tx, req.Email, req.NewPassword)
  if err == gorm.ErrRecordNotFound {
    rollbackTx(tx)
    return nil, status.Error(codes.Unauthenticated, "An account does not exist with this email.")
  }
  resp := &rpb.EditAccountPasswordResponse{}
  log.Printf(":EditAccountPassword: resp=%v", resp)
  return resp, nil
}


// CreateRoom creates a new Room (strict or non-strict), sets the requesting
// user as host, and builds a new RoomUser.
func (r *roomzApiService) CreateRoom(ctx context.Context, req *rpb.CreateRoomRequest) (*rpb.CreateRoomResponse, error) {
  log.Printf(":CreateRoom: Received data=%v", req)
  resp := &rpb.CreateRoomResponse{}
  tx := r.RDB.Begin()

  if len(req.UserName) == 0 {
    rollbackTx(tx)
    return nil, status.Error(codes.Unauthenticated, "Invalid username, must be nonempty.")
  }
  if len(req.Password) == 0 {
    rollbackTx(tx)
    return nil, status.Error(codes.Unauthenticated, "Invalid room password, must be nonempty.")
  }
  room := models.Room{
    HostId: int(req.UserId),
    HostSid: "0",
    Password: req.Password,
    CreationDate: time.Now(),
    IsStrict: req.IsStrict,
    UserLimit: maxUsersInRoom,
    IsActive: true,
  }
  var err error
  if err = r.RDB.CreateRoom(tx, &room); err != nil {
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, "Failed to create room.")
  }
  log.Printf(":CreateRoom: Created new room. Id=%v", room.Id)
  // TODO: Convert userId to type uint32.
  userIdStr := strconv.Itoa(int(req.UserId))
  if _, err = r.RDB.GetUser(tx, userIdStr); err == gorm.ErrRecordNotFound {
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, "Failed to find user!")
  }
  roomUser := models.RoomUser{
    RoomId:   room.Id,
    UserId:   int(req.UserId),
    Username: req.UserName,
  }
  if err := r.RDB.CreateRoomUser(tx, &roomUser); err != nil {
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, err.Error())
  }
  log.Printf(":CreateRoom: Created new RoomUser: %v", roomUser)

  tkn := r.createRoomToken(uint32(room.Id), req.UserId)
  resp = &rpb.CreateRoomResponse{
    Success: true,
    RoomId: uint32(room.Id),
    IsStrict: req.IsStrict,
    Token: tkn,
    ErrorMessage: "",
  }
  commitTx(tx)
  log.Printf(":CreateRoom: resp=%v", resp)
  return resp, nil
}


// AwaitRoomClosure opens a RoomzApiService_AwaitRoomClosure gRPC stream for
// a roomId and userId to await room closure by the host.
func (r *roomzApiService) AwaitRoomClosure(req *rpb.AwaitRoomClosureRequest, closeStream rpb.RoomzApiService_AwaitRoomClosureServer) (error) {
  log.Printf(":AwaitRoomClosure: Received data=%v", req)
  if !r.verifyToken(req.GetToken()) {
    return status.Error(codes.Unauthenticated, "invalid token")
  }

  closeChannel := make(chan *rpb.HostClosedRoom)
  userId := req.GetUserId()
  userCloseChannel := roomUserCloseChannel{userId: userId, channel: closeChannel}
  r.closeStreamsMtx.Lock()
  r.RoomCloseStreams[req.RoomId] = append(r.RoomCloseStreams[req.RoomId], userCloseChannel)
  r.closeStreamsMtx.Unlock()

  // TODO: On a HostClosedRoom message, do I need to send some sort of signal to chatChannel to end the stream properly?
  for {
    select {
    case <-closeStream.Context().Done():
      return nil
    case msg := <-closeChannel:
      closeStream.Send(msg)
    }
  }
}


// CloseRoom checks if a room exists and is active, and if so sets as inactive,
// emits a message on the RoomClosure gRPC stream, and then closes all room
// streams.
func (r *roomzApiService) CloseRoom(ctx context.Context, req *rpb.CloseRoomRequest) (*rpb.CloseRoomResponse, error) {
  log.Printf(":CloseRoom: Received data=%v", req)
  resp := &rpb.CloseRoomResponse{}
  tx := r.RDB.Begin()

  roomId := req.GetRoomId()
  var err error
  var room models.Room
  if room, err = r.RDB.GetRoom(tx, fmt.Sprintf("%v", roomId)); err == gorm.ErrRecordNotFound {
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, "Room does not exist!")
  }

  if !room.IsActive {
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, "Room is already closed!")
  }
  room.IsActive = false
  if tx != nil {
    tx.Save(&room)
    tx.Commit()
  }
  
  resp = &rpb.CloseRoomResponse{
    Success: true,
    ErrorMessage: "",
  }
  r.broadcastHostClosedRoom(roomId, &rpb.HostClosedRoom{})
  r.deleteRoomStreams(roomId)
  log.Printf(":CloseRoom: Emitting HostClosedRoom message to all RoomCloseStreams")
  log.Printf(":CloseRoom: resp=%v", resp)
  return resp, nil
}


// JoinRoom handles users joining strict or non-strict rooms. If strict, a
// JoinRoomRequest object is sent to the host. If not, a new RoomUser is
// created with the room's current chat history sent back to the requestor. 
func (r *roomzApiService) JoinRoom(req *rpb.JoinRoomRequest, joinRoomStream rpb.RoomzApiService_JoinRoomServer) (error) {
  log.Printf(":JoinRoom: Received data=%v", req)
  // TODO: Break this function into smaller pieces.
  resp := &rpb.JoinRoomResponse{}
  tx := r.RDB.Begin()

  var err error
  roomId := req.GetRoomId()
  roomPassword := req.GetRoomPassword()
  userId := req.GetUserId()
  userName := req.GetUserName()
  isGuest := req.GetIsGuest()

  var room models.Room
  if room, err = r.RDB.GetRoom(tx, fmt.Sprintf("%v", roomId)); err == gorm.ErrRecordNotFound {
    resp.Status = "reject"
    return status.Error(codes.Internal, "Room does not exist!")
  }

  if room.Password != roomPassword {
    rollbackTx(tx)
    return status.Error(codes.Unauthenticated, "Incorrect Room Password!")
  }

  if !room.IsActive {
    rollbackTx(tx)
    return status.Error(codes.Internal, "Room is no longer active!")
  }

  resp = &rpb.JoinRoomResponse{
    Success:      true,
    RoomId:       uint32(room.Id),
    UserId:       userId,
    IsGuest:      false,
    Status:       "accept",
    ErrorMessage: "",
  }
  
  serverError := false
  var user models.User
  if isGuest {
    // Create a new User for the guest.
    if serverError = r.RDB.CreateUser(tx, &user) != nil; serverError {
      rollbackTx(tx)
      return status.Error(codes.Internal, "Internal server error.")
    }
    log.Printf(":JoinRoom: Created new User=%v", user)
    
    // Create a Guest.
    // TODO: Is the Guest object needed?
    var guest models.Guest
    guest.UserId = user.Id
    if serverError = r.RDB.CreateGuest(tx, &guest) != nil; serverError {
      rollbackTx(tx)
      return status.Error(codes.Internal, "Internal server error.")
    }
    log.Printf(":JoinRoom: Created new Guest=%v", guest)

    resp.IsGuest = true
    resp.UserId = uint32(user.Id)
  } else {
    // uncover user from database
    if user, err = r.RDB.GetUser(tx, fmt.Sprintf("%v", userId)); err == gorm.ErrRecordNotFound {
      rollbackTx(tx)
      return status.Error(codes.Internal, "Internal server error.")
    }
  }

  if room.IsStrict {
    resp.Status = "wait"
    // TODO: Eliminate session ID.
    roomJoinRequest := models.RoomJoinRequest{
      RoomId:    room.Id,
      UserId:    user.Id,
      Username:  userName,
      SessionId: "",
    }
    if err = r.RDB.CreateRoomJoinRequest(tx, &roomJoinRequest); err != nil {
      rollbackTx(tx)
      return status.Error(codes.Internal, err.Error())
    }
    log.Printf(":JoinRoom: Created RoomJoinRequest=%v", roomJoinRequest)
  } else {
    // In a Non-strict room.

    curTime := time.Now()
    roomUser := models.RoomUser{
      RoomId:   room.Id,
      UserId:   user.Id,
      Username: userName,
      JoinDate: curTime,
    }
    if err = r.RDB.CreateRoomUser(tx, &roomUser); err != nil {
      rollbackTx(tx)
      return status.Error(codes.Internal, "Internal server error.")
    }

    // Log an activity message for the event.
    activityMessageString := fmt.Sprintf("%v has joined the room", roomUser)
    activityMessage := models.Message{
      RoomId:        room.Id,
      UserId:        user.Id,
      Username:      userName,
      Message:       activityMessageString,
      Timestamp:     curTime,
      MessageTypeId: activityMessageTypeId,
    }
    if r.RDB.CreateMessage(tx, &activityMessage); err != nil {
      rollbackTx(tx)
      return status.Error(codes.Internal, "Internal server error.")
    }
    // Add current chat room history to the response.
    roomChatHistory := r.RDB.GetRoomChatMessages(tx, fmt.Sprintf("%v", roomId))
    log.Printf(":JoinRoom: Retrieved chatroom history=%v", roomChatHistory)
    resp.ChatHistory = roomChatHistory
    resp.Token = r.createRoomToken(uint32(room.Id), userId)
  }
  commitTx(tx)
  
  // Create join room stream if not already created.
  joinRoomChannel := make(chan *rpb.JoinRoomResponse)
  joinKey := fmt.Sprintf("%v-%v", roomId, resp.GetUserId())

  r.joinStreamsMtx.Lock()
  r.RoomJoinStreams[joinKey] = joinRoomChannel
  r.joinStreamsMtx.Unlock()

  // Emit response on the JoinRoom gRPC stream.
  log.Printf(":JoinRoom: resp=%v", resp)
  go func() { joinRoomChannel <- resp }()

  for {
    select {
    case <-joinRoomStream.Context().Done():
      return nil
    case msg := <-joinRoomChannel:
      log.Print("Received on joinRoomChannel=%v", msg)
      joinRoomStream.Send(msg)
    }
  }
}


// LeaveRoom deletes a RoomUser from a valid room.
func (r *roomzApiService) LeaveRoom(ctx context.Context, req *rpb.LeaveRoomRequest) (*rpb.LeaveRoomResponse, error) {
  log.Printf(":LeaveRoom: Received data=%v", req)
  resp := &rpb.LeaveRoomResponse{}
  tx := r.RDB.Begin()

  var err error
  roomId := req.GetRoomId()
  roomIdStr := fmt.Sprintf("%v", roomId)
  if _, err = r.RDB.GetRoom(tx, roomIdStr); err == gorm.ErrRecordNotFound {
    rollbackTx(tx)
    return nil, status.Errorf(codes.Internal, "Room Id does not exist")
  }
  userId := req.GetUserId()
  userIdStr := fmt.Sprintf("%v", userId)

  var roomUser models.RoomUser
  if roomUser, err = r.RDB.GetRoomUser(tx, roomIdStr, userIdStr); err == gorm.ErrRecordNotFound {
    rollbackTx(tx)
    return nil, status.Errorf(codes.Internal, "Could not find user in room")
  }
  if tx != nil {
    tx.Delete(&roomUser)
    tx.Commit()
  } else if r.RDB.Testmode {
    r.RDB.DeleteRoomUser(&roomUser)
  }
  resp.Success = true
  log.Printf(":LeaveRoom: resp=%v", resp)
  return resp, nil
}


// EnterChatRoom opens a fresh EnterChatRoom gRPC stream for a user to receive
// chat room messages on.
func (r *roomzApiService) EnterChatRoom(req *rpb.EnterChatRoomRequest, chatStream rpb.RoomzApiService_EnterChatRoomServer) (error) {
  log.Printf(":EnterChatRoom: Received data=%v", req)
  if !r.verifyToken(req.GetToken()) {
    return status.Error(codes.Unauthenticated, "invalid token")
  }

  chatChannel := make(chan *rpb.ChatMessage)
  userId := req.GetUserId()
  userChatChannel := roomUserChatChannel{userId: userId, channel: chatChannel}

  r.chatStreamsMtx.Lock()
  r.RoomChatStreams[req.RoomId] = append(r.RoomChatStreams[req.RoomId], userChatChannel)
  r.chatStreamsMtx.Unlock()

  // TODO: on a HostClosedRoom message, do I need to send some sort of signal to chatChannel to end the stream properly?
  for {
    select {
    case <-chatStream.Context().Done():
      return nil
    case msg := <-chatChannel:
      chatStream.Send(msg)
    }
  }
}


// SendChatMessage creates a new Message object which is sent on the Room chat
// channel.
func (r *roomzApiService) SendChatMessage(ctx context.Context, req *rpb.ChatMessage) (*rpb.SendChatMessageResponse, error) {
  log.Printf(":CreateAccount: Received data=%v", req)
  resp := &rpb.SendChatMessageResponse{}
  tx := r.RDB.Begin()

  var err error
  var room models.Room

  // TODO: Edit GetRoom() calls to use standard uint32 type.
  roomId := fmt.Sprintf("%v", req.GetRoomId())
  userId := fmt.Sprintf("%v", req.GetUserId())
  if room, err = r.RDB.GetRoom(tx, roomId); err == gorm.ErrRecordNotFound {
    rollbackTx(tx)
    return nil, status.Error(codes.Unauthenticated, "room not found with associated room_id")
  }

  var roomUser models.RoomUser
  if roomUser, err = r.RDB.GetRoomUser(tx, roomId, userId); err == gorm.ErrRecordNotFound {
    errMsg := fmt.Sprintf("failed to find user with user_id=%v in room_id=%v", req.GetUserId(), req.GetRoomId())
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, errMsg)
  }

  messageTimestamp := time.Now()
  chatroomMessage := models.Message{
    RoomId:        room.Id,
    UserId:        roomUser.UserId,
    Username:      roomUser.Username,
    Message:       req.GetMessage(),
    Timestamp:     messageTimestamp,
    MessageTypeId: chatroomMessageTypeId,
  }
  if err = r.RDB.CreateMessage(tx, &chatroomMessage); err != nil {
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, err.Error())
  }

  resp = &rpb.SendChatMessageResponse{
    Success: true,
    ErrorMessage: "",
  }
  commitTx(tx)

  log.Printf(":SendChatMessage: Broadcasting chat to room")
  // Emit the chat message on the room chat channel.
  go r.broadcastChatToRoom(req)
  return resp, nil
}


// GetJoinRequests returns all RoomJoinRequests for a given roomId.
func (r *roomzApiService) GetJoinRequests(ctx context.Context, req *rpb.GetJoinRequestsRequest) (*rpb.GetJoinRequestsResponse, error) {
  log.Printf(":CreateAccount: Received data=%v", req)
  resp := &rpb.GetJoinRequestsResponse{}
  tx := r.RDB.Begin()

  roomId := req.GetRoomId()

  var room models.Room
  var err error
  if room, err = r.RDB.GetRoom(tx, fmt.Sprintf("%v", roomId)); err == gorm.ErrRecordNotFound {
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to find room with room id: %v", roomId))
  }
  joinRequests := r.RDB.GetRoomJoinRequests(tx, room.Id)
  resp.Success = true
  resp.RoomId = roomId
  resp.JoinRequests = joinRequests
  commitTx(tx)

  log.Printf(":GetJoinRequests: Emitting response=%v", resp)
  return resp, nil
}


// HandleJoinRequest makes a decision to accept or reject a given
// RoomJoinRequest object, and sends the response to the requesting user.
func (r *roomzApiService) HandleJoinRequest(ctx context.Context, req *rpb.HandleJoinRequestRequest) (*rpb.HandleJoinRequestResponse, error) {
  log.Printf(":CreateAccount: Received data=%v", req)
  resp := &rpb.HandleJoinRequestResponse{}
  tx := r.RDB.Begin()

  roomId := req.GetRoomId()
  userIdToHandle := req.GetUserIdToHandle()
  decision := req.GetDecision()
  if decision != "accept" && decision != "reject" {
    return nil, status.Error(codes.Unauthenticated, "Invalid decision. Must be: [accept|reject]")
  }

  var room models.Room
  var err error
  roomIdStr := fmt.Sprintf("%v", roomId)
  userIdToHandleStr := fmt.Sprintf("%v", userIdToHandle)
  if room, err = r.RDB.GetRoom(tx, roomIdStr); err == gorm.ErrRecordNotFound {
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to find room with room id: %v", roomId))
  }
  var roomJoinRequest models.RoomJoinRequest
  if roomJoinRequest, err = r.RDB.GetRoomJoinRequest(tx, roomIdStr, userIdToHandleStr); err == gorm.ErrRecordNotFound {
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, "No join request found!")
  }

  joinRoomResp := &rpb.JoinRoomResponse{}
  if acceptUser := decision == "accept"; acceptUser {
    roomChatHistory := r.RDB.GetRoomChatMessages(tx, roomIdStr)
    log.Printf(":HandleJoinRequest: retrieved chatroom history=%v", roomChatHistory)

    // Verify this user still exists.
    var user models.User
    if user, err = r.RDB.GetUser(tx, userIdToHandleStr); err == gorm.ErrRecordNotFound {
      return nil, status.Error(codes.Internal, "User no longer exists!")
    }

    userName := roomJoinRequest.Username
    newRoomUser := models.RoomUser{
      RoomId:   room.Id,
      UserId:   user.Id,
      Username: userName,
    }
    r.RDB.CreateRoomUser(tx, &newRoomUser)
    log.Printf(":handleJoinRequest: Created new RoomUser=%v", newRoomUser)

    // Log new ActivityMessage if user is joining the room.
    activityMessageString := fmt.Sprintf("%v has joined the room", newRoomUser)
    activityMessage := models.Message{
      RoomId:        room.Id,
      UserId:        user.Id,
      Username:      userName,
      Message:       activityMessageString,
      Timestamp:     time.Now(),
      MessageTypeId: activityMessageTypeId,
    }
    if err = r.RDB.CreateMessage(tx, &activityMessage); err != nil {
      panic("Internal server error.")
    }
    log.Printf(":HandleJoinRequest: Created new AcitivityMessage=%v", activityMessage)
    roomToken := r.createRoomToken(roomId, userIdToHandle)

    joinRoomResp = &rpb.JoinRoomResponse{
      Success:      true,
      RoomId:       roomId,
      Token:        roomToken,
      UserId:       userIdToHandle,
      Status:       "accept",
      ChatHistory:  roomChatHistory,
      ErrorMessage: "",
    }
  } else {
    joinRoomResp = &rpb.JoinRoomResponse{
      Success: true,
      RoomId:  roomId,
      UserId:  userIdToHandle,
      Status:  "reject",
    }
  }
  // No matter the decision, delete the RoomJoinRequest.
  if tx != nil {
    tx.Delete(&roomJoinRequest)
    tx.Commit()
  } else if r.RDB.Testmode {
    r.RDB.DeleteRoomJoinRequest(&roomJoinRequest)
  }
  // Send response on the JoinRoom channel.
  joinKey := fmt.Sprintf("%v-%v", roomId, userIdToHandle)
  r.joinStreamsMtx.Lock()
  joinChannel := r.RoomJoinStreams[joinKey]
  r.joinStreamsMtx.Unlock()

  go func() { joinChannel <- joinRoomResp }()

  if joinRoomResp.GetSuccess() {
    log.Printf(":HandleJoinRequest: sending message to host...")
  }
  resp = &rpb.HandleJoinRequestResponse{
    Success: joinRoomResp.GetSuccess(),
    ErrorMessage: joinRoomResp.GetErrorMessage(),
  }
  return resp, nil
}


// CancelJoinRequest deletes a user's RoomJoinRequest.
func (r *roomzApiService) CancelJoinRequest(ctx context.Context, req *rpb.CancelJoinRequestRequest) (*rpb.CancelJoinRequestResponse, error) {
  log.Printf(":CreateAccount: Received data=%v", req)
  resp := &rpb.CancelJoinRequestResponse{}
  tx := r.RDB.Begin()

  roomId := req.GetRoomId()
  roomIdStr := fmt.Sprintf("%v", roomId)
  userId := req.GetUserId()
  userIdStr := fmt.Sprintf("%v", userId)

  var err error
  if _, err = r.RDB.GetRoom(tx, roomIdStr); err == gorm.ErrRecordNotFound {
    return nil, status.Error(codes.Internal, "Room does not exist!")
  }
  var roomJoinRequest models.RoomJoinRequest
  if roomJoinRequest, err = r.RDB.GetRoomJoinRequest(tx, roomIdStr, userIdStr); err == gorm.ErrRecordNotFound {
    return nil, status.Error(codes.Internal, "Room Join Request does not exist!")
  }
  if tx != nil {
    tx.Delete(&roomJoinRequest)
  } else if r.RDB.Testmode {
    r.RDB.DeleteRoomJoinRequest(&roomJoinRequest)
  }
  log.Printf(":CancelJoinRequest: Delete join request successfully")
  resp.Success = true
  commitTx(tx)
  return resp, nil
}


// UpdateSessionId is not implemented. This is a socket.io concept.
// TODO: Delete.
func (r *roomzApiService) UpdateSessionId(ctx context.Context, req *rpb.UpdateSessionIdRequest) (*rpb.UpdateSessionIdResponse, error) {
  log.Printf(":CreateAccount: Received data=%v", req)
  // TODO: this is only necessary to build out when we start working with the FE
  return &rpb.UpdateSessionIdResponse{
    Success: true,
    ErrorMessage: "",
  }, nil
}

// End of public api.

func (r *roomzApiService) deleteUserRoomStreams(roomId, userId uint32) {
  log.Printf(":deleteUserRoomStreams:")
  r.chatStreamsMtx.Lock()
  for i, userChatChannel := range r.RoomChatStreams[roomId] {
    if userChatChannel.userId == userId {
      r.RoomChatStreams[roomId] = append(r.RoomChatStreams[roomId][:i], r.RoomChatStreams[roomId][i+1:]...)
      break
    }
  }
  r.chatStreamsMtx.Unlock()

  r.closeStreamsMtx.Lock()
  for i, userCloseChannel := range r.RoomCloseStreams[roomId] {
    if userCloseChannel.userId == userId {
      r.RoomCloseStreams[roomId] = append(r.RoomCloseStreams[roomId][:i], r.RoomCloseStreams[roomId][i+1:]...)
      break
    }
  }
  r.closeStreamsMtx.Unlock()

  r.joinStreamsMtx.Lock()
  delete(r.RoomJoinStreams, fmt.Sprintf("%v-%v", roomId, userId))
  r.joinStreamsMtx.Unlock()
}

func (r *roomzApiService) deleteRoomStreams(roomId uint32) {
  log.Printf(":deleteRoomStreams:")

  // Delete chat streams.
  r.chatStreamsMtx.Lock()
  delete(r.RoomChatStreams, roomId)
  r.chatStreamsMtx.Unlock()

  // Delete room closure streams.
  r.closeStreamsMtx.Lock()
  delete(r.RoomCloseStreams, roomId)
  r.closeStreamsMtx.Unlock()
}

func (r *roomzApiService) broadcastHostClosedRoom(roomId uint32, closeMsg *rpb.HostClosedRoom) {
  log.Printf(":broadcastHostClosedRoom:")
  wait := sync.WaitGroup{}
  done := make(chan int)

  r.closeStreamsMtx.Lock()
  roomCloseChannels := r.RoomCloseStreams[roomId]
  r.closeStreamsMtx.Unlock()
  
  for _, roomCloseChannel := range roomCloseChannels {
    wait.Add(1)
    go func(msg *rpb.HostClosedRoom, msgChan chan *rpb.HostClosedRoom) {
      defer wait.Done()
      msgChan <- msg
    }(closeMsg, roomCloseChannel.channel)
  }

  go func() {
    wait.Wait()
    close(done)
  }()
  <-done
}

func (r *roomzApiService) broadcastChatToRoom(msg *rpb.ChatMessage) {
  log.Printf(":broadcastChatToRoom:")
  wait := sync.WaitGroup{}
  done := make(chan int)

  r.chatStreamsMtx.Lock()
  userChatChannels := r.RoomChatStreams[msg.GetRoomId()]
  r.chatStreamsMtx.Unlock()

  for _, userChatChannel := range userChatChannels {
    wait.Add(1)
    go func(msg *rpb.ChatMessage, msgChan chan *rpb.ChatMessage) {
      defer wait.Done()
      msgChan <- msg
    }(msg, userChatChannel.channel)
  }

  go func() {
    wait.Wait()
    close(done)
  }()
  <-done
}

func (r *roomzApiService) createRoomToken(roomId, userId uint32) string {
  randTkn := make([]byte, 12)
  rand.Read(randTkn)
  
  tkn := fmt.Sprintf("%x-%v-%v", randTkn, roomId, userId)
  r.ActiveTkns = append(r.ActiveTkns, tkn)
  return tkn
}

func (r *roomzApiService) verifyToken(tkn string) bool {
  r.tknsMtx.Lock()
  activeTkns := r.ActiveTkns
  r.tknsMtx.Unlock()

  for _, t := range activeTkns {
    if t == tkn {
      return true
    }
  }
  return false
}