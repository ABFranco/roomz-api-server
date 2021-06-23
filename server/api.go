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

  // check if account already exists
  account, res := r.RDB.GetAccount(tx, req.Email)
  if res != gorm.ErrRecordNotFound {
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, "An account already exists with this email!")
  }
  
  // create user
  var user models.User
  fullName := fmt.Sprintf("%s %s", req.FirstName, req.LastName)
  user.Name.String = fullName
  if err := r.RDB.CreateUser(tx, &user); err != nil {
    log.Printf(":CreateAccount: Failed to create User. Err=%v", err)
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, "Internal server error.")
  }

  // create account
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
  // TODO: make user.Id uint32, this can cause overflow if negative
  resp.UserId = uint32(user.Id)
  commitTx(tx)

  log.Printf(":CreateAccount: resp=%v", resp)
  return resp, nil
}


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
  // check if account exists with email
  var account models.Account
  var err error
  if account, err = r.RDB.GetAccount(tx, req.Email);  err == gorm.ErrRecordNotFound {
    rollbackTx(tx)
    return nil, status.Error(codes.Unauthenticated, "An account does not exist with this email.")
  }

  // verify password
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

// TODO: Add additional security here and in EditAccountPassword in the future.
// The email will be validated in the frontend 
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

func (r *roomzApiService) EditAccountPassword(ctx context.Context, req *rpb.EditAccountPasswordRequest) (*rpb.EditAccountPasswordResponse, error) {
  log.Printf(":EditAccountPassword: Received data=%v", req)
  tx := r.RDB.Begin()
  // if user doesn't enter anything for email and/or textbox 
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
  //wrong password: check if err == "the error that indicates an incorrect password"
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
  isStrict := req.IsStrict

  // create room
  room := models.Room{
    HostId: int(req.UserId),
    HostSid: "0",
    Password: req.Password,
    CreationDate: time.Now(),
    IsStrict: isStrict,
    UserLimit: maxUsersInRoom,
    IsActive: true,
  }
  var err error
  if err = r.RDB.CreateRoom(tx, &room); err != nil {
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, "Failed to create room.")
  }
  log.Printf(":CreateRoom: Created new room. Id=%v", room.Id)

  // retrieve user
  // TEMP HACK
  userIdStr := strconv.Itoa(int(req.UserId))
  if _, err = r.RDB.GetUser(tx, userIdStr); err == gorm.ErrRecordNotFound {
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, "Failed to find user!")
  }

  // create new RoomUser
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
    IsStrict: isStrict,
    Token: tkn,
    ErrorMessage: "",
  }

  commitTx(tx)
  log.Printf(":CreateRoom: resp=%v", resp)
  return resp, nil
}


func (r *roomzApiService) AwaitRoomClosure(req *rpb.AwaitRoomClosureRequest, closeStream rpb.RoomzApiService_AwaitRoomClosureServer) (error) {
  log.Printf(":AwaitRoomClosure: Received data=%v", req)
  if !r.verifyToken(req.GetToken()) {
    return status.Error(codes.Unauthenticated, "invalid token")
  }

  // create new chat message channel associated with room Id
  closeChannel := make(chan *rpb.HostClosedRoom)
  userId := req.GetUserId()
  userCloseChannel := roomUserCloseChannel{userId: userId, channel: closeChannel}
  r.closeStreamsMtx.Lock()
  r.RoomCloseStreams[req.RoomId] = append(r.RoomCloseStreams[req.RoomId], userCloseChannel)
  r.closeStreamsMtx.Unlock()

  // TODO: on a HostClosedRoom message, do I need to send some sort of signal to chatChannel to end the stream properly?
  for {
    select {
    case <-closeStream.Context().Done():
      return nil
    case msg := <-closeChannel:
      closeStream.Send(msg)
    }
  }
}


func (r *roomzApiService) CloseRoom(ctx context.Context, req *rpb.CloseRoomRequest) (*rpb.CloseRoomResponse, error) {
  log.Printf(":CloseRoom: Received data=%v", req)
  resp := &rpb.CloseRoomResponse{}
  tx := r.RDB.Begin()

  var err error
  roomId := req.GetRoomId()

  // retreive room to close
  var room models.Room
  if room, err = r.RDB.GetRoom(tx, fmt.Sprintf("%v", roomId)); err == gorm.ErrRecordNotFound {
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, "Room does not exist!")
  }

  // check if room is inactive
  if !room.IsActive {
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, "Room is already closed!")
  }

  // mark room as inactive (closed)
  room.IsActive = false
  if tx != nil {
    tx.Save(&room)
    tx.Commit()
  }
  
  resp = &rpb.CloseRoomResponse{
    Success: true,
    ErrorMessage: "",
  }

  // send room closure notification to all room streams
  r.broadcastHostClosedRoom(roomId, &rpb.HostClosedRoom{})
  r.deleteRoomStreams(roomId)
  log.Printf(":CloseRoom: Emitting HostClosedRoom message to all RoomCloseStreams")
  log.Printf(":CloseRoom: resp=%v", resp)
  return resp, nil
}

func (r *roomzApiService) JoinRoom(req *rpb.JoinRoomRequest, joinRoomStream rpb.RoomzApiService_JoinRoomServer) (error) {
  log.Printf(":JoinRoom: Received data=%v", req)
  // TODO: break this function into smaller pieces
  resp := &rpb.JoinRoomResponse{}
  tx := r.RDB.Begin()

  var err error
  roomId := req.GetRoomId()
  roomPassword := req.GetRoomPassword()
  userId := req.GetUserId()
  userName := req.GetUserName()
  isGuest := req.GetIsGuest()

  // check if room exists
  var room models.Room
  if room, err = r.RDB.GetRoom(tx, fmt.Sprintf("%v", roomId)); err == gorm.ErrRecordNotFound {
    resp.Status = "reject"
    return status.Error(codes.Internal, "Room does not exist!")
  }

  // verify passwords match
  if room.Password != roomPassword {
    rollbackTx(tx)
    return status.Error(codes.Unauthenticated, "Incorrect Room Password!")
  }

  // verify room is active
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
    // create fresh user
    if serverError = r.RDB.CreateUser(tx, &user) != nil; serverError {
      rollbackTx(tx)
      return status.Error(codes.Internal, "Internal server error.")
    }
    log.Printf(":JoinRoom: Created new User=%v", user)
    
    // create guest
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

  // handle strict room
  if room.IsStrict {
    resp.Status = "wait"

    // add user to pending join requests for room
    // TODO: eliminate session ID
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

    // NOTE: old implementation sent "incomingJoinRequest" notification to host. Keep it live?
  } else {
    // non-strict room

    // create new RoomUser
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

    // create activity message
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
    // retrieve chatroom messages
    roomChatHistory := r.RDB.GetRoomChatMessages(tx, fmt.Sprintf("%v", roomId))
    log.Printf(":JoinRoom: Retrieved chatroom history=%v", roomChatHistory)
	resp.ChatHistory = roomChatHistory
	resp.Token = r.createRoomToken(uint32(room.Id), userId)
  }
  commitTx(tx)
  
  // create join room stream if not already created
  joinRoomChannel := make(chan *rpb.JoinRoomResponse)
  joinKey := fmt.Sprintf("%v-%v", roomId, resp.GetUserId())

  r.joinStreamsMtx.Lock()
  r.RoomJoinStreams[joinKey] = joinRoomChannel
  r.joinStreamsMtx.Unlock()

  // send response to stream
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


func (r *roomzApiService) EnterChatRoom(req *rpb.EnterChatRoomRequest, chatStream rpb.RoomzApiService_EnterChatRoomServer) (error) {
  log.Printf(":EnterChatRoom: Received data=%v", req)
  if !r.verifyToken(req.GetToken()) {
    return status.Error(codes.Unauthenticated, "invalid token")
  }

  // create new chat message channel associated with room Id
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


func (r *roomzApiService) SendChatMessage(ctx context.Context, req *rpb.ChatMessage) (*rpb.SendChatMessageResponse, error) {
  log.Printf(":CreateAccount: Received data=%v", req)
  resp := &rpb.SendChatMessageResponse{}
  tx := r.RDB.Begin()

  var err error
  var room models.Room

  // TODO: edit GetRoom() calls to use standard uint32 type
  roomId := fmt.Sprintf("%v", req.GetRoomId())
  userId := fmt.Sprintf("%v", req.GetUserId())
  if room, err = r.RDB.GetRoom(tx, roomId); err == gorm.ErrRecordNotFound {
    rollbackTx(tx)
    return nil, status.Error(codes.Unauthenticated, "room not found with associated room_id")
  }

  // retrieve user from room
  var roomUser models.RoomUser
  if roomUser, err = r.RDB.GetRoomUser(tx, roomId, userId); err == gorm.ErrRecordNotFound {
    errMsg := fmt.Sprintf("failed to find user with user_id=%v in room_id=%v", req.GetUserId(), req.GetRoomId())
    rollbackTx(tx)
    return nil, status.Error(codes.Internal, errMsg)
  }

  // create message object
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
  go r.broadcastChatToRoom(req)
  return resp, nil
}

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
  // TODO: temp hack
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

    // TODO: make function both JoinRoom() and GetJoinRequests() use here.
    // check if user exists anymore
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

    // create activity message
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
  // delete remnant RoomJoinRequest
  if tx != nil {
    tx.Delete(&roomJoinRequest)
    tx.Commit()
  } else if r.RDB.Testmode {
    r.RDB.DeleteRoomJoinRequest(&roomJoinRequest)
  }
  // emit response to RoomJoinChannel
  joinKey := fmt.Sprintf("%v-%v", roomId, userIdToHandle)
  r.joinStreamsMtx.Lock()
  joinChannel := r.RoomJoinStreams[joinKey]
  r.joinStreamsMtx.Unlock()

  go func() { joinChannel <- joinRoomResp }()

  if joinRoomResp.GetSuccess() {
    // TODO: send fresh emit getJoinRequests event to host
    log.Printf(":HandleJoinRequest: sending message to host...")
  }

  resp = &rpb.HandleJoinRequestResponse{
    Success: joinRoomResp.GetSuccess(),
    ErrorMessage: joinRoomResp.GetErrorMessage(),
  }
  return resp, nil
}

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

  // TODO: send live event to host that there's a cancel coming in (not necessary)
  return resp, nil
}

func (r *roomzApiService) UpdateSessionId(ctx context.Context, req *rpb.UpdateSessionIdRequest) (*rpb.UpdateSessionIdResponse, error) {
  log.Printf(":CreateAccount: Received data=%v", req)
  // TODO: this is only necessary to build out when we start working with the FE
  return &rpb.UpdateSessionIdResponse{
    Success: true,
    ErrorMessage: "",
  }, nil
}

// end of public api

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

  // delete chat streams
  r.chatStreamsMtx.Lock()
  delete(r.RoomChatStreams, roomId)
  r.chatStreamsMtx.Unlock()

  // delete room closure streams
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

//
//  Sign-in/Create Account Handlers
//
// func (rs *RoomzServer) CreateAccountHandler(s socketio.Conn, data map[string]interface{}) {
//   log.Printf(":CreateAccount: Received data %v", data)

//   var err error
//   sioRoom := s.ID()
//   respEvent := "createAccountResp"
//   resp := map[string]interface{}{
//     "success":       false,
//     "user_id":       nil,
//     "error_message": nil,
//   }

//   // open fresh session
//   tx := rs.RDB.Begin()
//   func() {
//     defer eventError(resp, tx)
    
//     var firstName, lastName, email, password string
//     var ok bool
//     if firstName, ok = data["firstName"].(string); !ok || len(firstName) == 0 {
//       panic("Invalid firstName.")
//     }
//     if lastName, ok = data["lastName"].(string); !ok || len(lastName) == 0 {
//       panic("Invalid lastName.")
//     }
//     if email, ok = data["email"].(string); !ok || len(email) == 0 {
//       panic("Invalid email.")
//     }
//     if password, ok = data["password"].(string); !ok || len(password) == 0 {
//       panic("Invalid password.")
//     }

//     // check if account already exists
//     account, res := rs.RDB.GetAccount(tx, email)
//     if res != gorm.ErrRecordNotFound {
//       panic("An already exists with this email!")
//     }

//     // create user
//     var user models.User
//     fullName := fmt.Sprintf("%s %s", firstName, lastName)
//     user.Name.String = fullName
//     if err = rs.RDB.CreateUser(tx, &user); err != nil {
//       log.Printf(":CreateAccount: Failed to create User. Err=%v", err)
//       panic("Internal server error.")
//     }

//     // create account
//     account = models.Account{
//       UserId:    user.Id,
//       Email:     email,
//       Password:  util.Encrypt(password),
//       FirstName: firstName,
//       LastName:  lastName,
//       StartDate: time.Now(),
//     }

//     if err = rs.RDB.CreateAccount(tx, &account); err != nil {
//       log.Printf(":CreateAccount: Failed to create Account. Err=%v", err)
//       panic("Internal server error.")
//     }

//     resp["success"] = true
//     resp["user_id"] = user.Id
//     if tx != nil {
//       tx.Commit()
//     }
//   }()
  
//   log.Printf(":CreateAccount: Emitting response=%v to id=%v", resp, sioRoom)
//   s.Emit(respEvent, resp, sioRoom)
// }

/*
func (rs *RoomzServer) SignInHandler(s socketio.Conn, data map[string]interface{}) {
  log.Printf(":SignIn: Received data %v", data)

  var err error
  sioRoom := s.ID()
  respEvent := "signInResp"
  resp := map[string]interface{}{
    "success":       false,
    "user_id":       nil,
    "first_name":    nil,
    "last_name":     nil,
    "error_message": nil,
  }

  // open fresh session
  tx := rs.RDB.Begin()
  func() {
    defer eventError(resp, tx)

    var email, password string
    var ok bool
    if email, ok = data["email"].(string); !ok || len(email) == 0 {
      panic("Invalid email")
    }
    if password, ok = data["password"].(string); !ok || len(password) == 0  {
      panic("Invalid password")
    }
    
    // check if account exists with email
    var account models.Account
    if account, err = rs.RDB.GetAccount(tx, email); err == gorm.ErrRecordNotFound {
      panic("Invalid account details!")
    }

    // check for incorrect password
    if util.Decrypt(account.Password) != password {
      panic("Invalid password!")
    }

    // build response
    resp["success"] = true
    resp["user_id"] = account.Id
    resp["first_name"] = account.FirstName
    resp["last_name"] = account.LastName
  }()

  log.Printf(":SignIn: Emitting response=%v to id=%v", resp, sioRoom)
  s.Emit(respEvent, resp, sioRoom)
}

//
//  Room Handlers
//
func (rs *RoomzServer) CreateRoomHandler(s socketio.Conn, data map[string]interface{}) {
  log.Printf(":CreateRoom: Received data %v", data)

  hostSessionId := s.ID()
  sioRoom := s.ID()
  respEvent := "createRoomResp"
  resp := map[string]interface{}{
    "success":   true,
    "room_id":   0,
    "is_strict": false,
  }

  // open fresh session
  tx := rs.RDB.Begin()
  var room models.Room
  func() {
    defer eventError(resp, tx)
    
    var userId, username, roomPassword string
    var isStrict, ok bool

    userId = fmt.Sprintf("%v", data["user_id"])
    if username, ok = data["username"].(string); !ok || len(username) == 0 {
      panic("Invalid username.")
    }
    if roomPassword, ok = data["password"].(string); !ok || len(roomPassword) == 0 {
      panic("Invalid room password.")
    }
    if isStrict, ok = data["is_strict"].(bool); !ok {
      panic("Invalid is_strict type.")
    }

    userIdInt, err := strconv.Atoi(userId);
    if err != nil {
      panic("user_id must be a valid integer.")
    }

    room = models.Room{
      HostId:       userIdInt,
      HostSid:      hostSessionId,
      Password:     roomPassword,
      CreationDate: time.Now(),
      IsStrict:     isStrict,
      UserLimit:    maxUsersInRoom,
      IsActive:     true,
    }
  
    // create room
    if err = rs.RDB.CreateRoom(tx, &room); err != nil {
      panic("Failed to create room.")
    }
    log.Printf(":CreateRoom: Created new room. Id=%v", room.Id)
  
    // retrieve user
    if _, err = rs.RDB.GetUser(tx, userId); err == gorm.ErrRecordNotFound {
      panic("Failed to find user!")
    }
  
    // create new RoomUser
    roomUser := models.RoomUser{
      RoomId:   room.Id,
      UserId:   userIdInt,
      Username: username,
    }
    if err := rs.RDB.CreateRoomUser(tx, &roomUser); err != nil {
      panic(err.Error())
    }
  
    log.Printf(":CreateRoom: Created new RoomUser: %v", roomUser)
  
    // add user to socketio room
    roomIdStr := strconv.Itoa(room.Id)
    rs.Server.JoinRoom("/", roomIdStr, s)
    log.Printf(":CreateRoom: Added userId=%v to roomId=%v", userId, room.Id)

    resp["success"] = true
    resp["room_id"] = room.Id
    resp["is_strict"] = isStrict

    if tx != nil {
      tx.Commit()
    }
  }()

  log.Printf(":CreateRoom: Emitting \"%v\" with resp=%v room=%v", respEvent, resp, room.Id)
  s.Emit(respEvent, resp, sioRoom)
}


func (rs *RoomzServer) CloseRoomHandler(s socketio.Conn, data map[string]interface{}) {
  log.Printf(":CloseRoom: Received data %v", data)
  
  var err error
  sioRoom := s.ID()
  respEvent := "closeRoomResp"
  resp := map[string]interface{}{
    "success": false,
  }

  // open fresh session
  tx := rs.RDB.Begin()
  var roomId string
  func() {
    defer eventError(resp, tx)

    roomId = fmt.Sprintf("%v", data["room_id"])
    if _, err = strconv.Atoi(roomId); err != nil {
      panic("Invalid room_id. Must be an integer.")
    }

    // retreive room to close
    var room models.Room
    if room, err = rs.RDB.GetRoom(tx, roomId); err == gorm.ErrRecordNotFound {
      panic("Room does not exist!")
    }

    // check if room is inactive
    if !room.IsActive {
      panic("Room is already closed!")
    }

    // mark room as inactive (closed)
    room.IsActive = false
    if tx != nil {
      tx.Save(&room)
      tx.Commit()
    }
    resp["success"] = true
  }()

  log.Printf(":CloseRoom: Emitting \"closeRoomResp\" with resp=%v", resp)
  s.Emit(respEvent, resp, sioRoom)

  // disconnect all clients in the room
  if resp["success"].(bool) {
    if _, err := strconv.Atoi(roomId); err != nil || roomId == "" {
      log.Printf(":CloseRoom: invalid room id!")
      return
    }

    disconnectData := map[string]interface{}{
      "room_id": roomId,
    }
    log.Printf(":CloseRoom: Disconnecting all clients with \"disconnectAll\", data=%v", disconnectData)
    rs.Server.BroadcastToRoom("/", roomId, "disconnectAll", disconnectData)
  }
}


func (rs *RoomzServer) JoinRoomHandler(s socketio.Conn, data map[string]interface{}) {
  log.Printf(":JoinRoom: Received data %v from %v", data, s.ID())

  var err error
  sioRoom := s.ID()
  resp := map[string]interface{}{
    "success": false,
  }

  // open fresh session
  tx := rs.RDB.Begin()
  func() {
    defer eventError(resp, tx)

    var roomId, roomPassword, userId, username string
    var isGuest, ok bool

    roomId = fmt.Sprintf("%v", data["room_id"])
    if _, err = strconv.Atoi(roomId); err != nil {
      panic("Invalid room_id. Must be an integer.")
    }
    if roomPassword, ok = data["room_password"].(string); !ok {
      panic("Invalid room_password type.")
    }
    userId = fmt.Sprintf("%v", data["user_id"])
    if username, ok = data["username"].(string); !ok {
      panic("Invalid username type.")
    }
    if isGuest, ok = data["is_guest"].(bool); !ok {
      panic("Invalid is_guest type.")
    }

    // check if room exists
    var room models.Room
    if room, err = rs.RDB.GetRoom(tx, roomId); err == gorm.ErrRecordNotFound {
      resp["room_id"] = nil
      resp["status"] = "reject"
      panic("Room does not exist!")
    }

    // verify passwords match
    if room.Password != roomPassword {
      resp["room_id"] = nil
      panic("Incorrect Room Password!")
    }

    // verify room is active
    if !room.IsActive {
      resp["room_id"] = nil
      panic("Room is no longer active!")
    }

    resp = map[string]interface{}{
      "success":       true,
      "room_id":       room.Id,
      "user_id":       userId,
      "status":        "accept",
      "chat_history":  nil,
      "error_message": nil,
    }
    
    serverError := false
    var user models.User
    if isGuest {
      // create fresh user
      if serverError = rs.RDB.CreateUser(tx, &user) != nil; serverError {
        panic("Internal server error.")
      }
      log.Printf(":JoinRoom: Created new User=%v", user)
      
      // create guest
      var guest models.Guest
      guest.UserId = user.Id
      if serverError = rs.RDB.CreateGuest(tx, &guest) != nil; serverError {
        panic("Internal server error.")
      }
      log.Printf(":JoinRoom: Created new Guest=%v", guest)

      resp["is_guest"] = true
      resp["user_id"] = user.Id
    } else {
      // uncover user from database
      if user, err = rs.RDB.GetUser(tx, userId); err == gorm.ErrRecordNotFound {
        panic("Internal server error.")
      }
    }

    // handle strict room
    if room.IsStrict {
      resp["status"] = "wait"

      roomHostSid := room.HostSid
      hostResp := map[string]interface{}{
        "room_id": room.Id,
        "user_id": user.Id,
        "name":    username,
      }

      // add user to pending join requests for room
      roomJoinRequest := models.RoomJoinRequest{
        RoomId: room.Id,
        UserId: user.Id,
        Username: username,
        SessionId: s.ID(),
      }

      // create room join request
      if err = rs.RDB.CreateRoomJoinRequest(tx, &roomJoinRequest); err != nil {
        panic(err.Error())
      }
      log.Printf(":JoinRoom: Created RoomJoinRequest=%v", roomJoinRequest)

      log.Printf(":JoinRoom: Emitting \"incomingJoinRequest\", data=%v to host", hostResp)
      s.Emit("/", "incomingJoinRequest", hostResp, roomHostSid)
    } else {
      // non-strict room

      // create new RoomUser
      roomUser := models.RoomUser{
        RoomId:   room.Id,
        UserId:   user.Id,
        Username: username,
        JoinDate: time.Now(),
      }
      if err = rs.RDB.CreateRoomUser(tx, &roomUser); err != nil {
        panic("Internal server error.")
      }

      // create activity message
      activityMessageString := fmt.Sprintf("%v has joined the room", roomUser)
      activityMessage := models.Message{
        RoomId:        room.Id,
        UserId:        user.Id,
        Username:      username,
        Message:       activityMessageString,
        Timestamp:     time.Now(),
        MessageTypeId: activityMessageTypeId,
      }
      if rs.RDB.CreateMessage(tx, &activityMessage); err != nil {
        panic("Internal server error.")
      }
      // retrieve chatroom messages
      roomChatHistory := rs.RDB.GetRoomChatMessages(tx, roomId)
      log.Printf(":JoinRoom: Retrieved chatroom history=%v", roomChatHistory)
      resp["chat_history"] = roomChatHistory
    }

    // add user to socketio room
    log.Printf(":JoinRoom: Adding user id=%v to room id=%v", userId, roomId)
    rs.Server.JoinRoom("/", roomId, s)
    
    if tx != nil {
      tx.Commit()
    }
  }()

  log.Printf(":JoinRoom: Emitting \"joinRoomResp\" with resp=%v", resp)
  s.Emit("joinRoomResp", resp, sioRoom)
}

func (rs *RoomzServer) LeaveRoomHandler(s socketio.Conn, data map[string]interface{}) {
  log.Printf(":LeaveRoom: Received data %v", data)

  var err error
  sioRoom := s.ID()
  resp := map[string]interface{}{
    "success": true,
  }

  // open fresh session
  tx := rs.RDB.Begin()
  func() {
    defer eventError(resp, tx)

    roomId := fmt.Sprintf("%v", data["room_id"])
    if _, err = strconv.Atoi(roomId); err != nil {
      panic("Invalid room_id. Must be an integer.")
    }
    if _, err = rs.RDB.GetRoom(tx, roomId); err == gorm.ErrRecordNotFound {
      panic("room_id does not exist.")
    }
    
    userId := fmt.Sprintf("%v",  data["user_id"])
    if _, err = strconv.Atoi(userId); err != nil {
      panic("Invalid user_id. Must be an integer.")
    }
    if _, err = rs.RDB.GetUser(tx, userId); err == gorm.ErrRecordNotFound {
      panic("user_id does not exist.")
    }

    var roomUser models.RoomUser
    if roomUser, err = rs.RDB.GetRoomUser(tx, roomId, userId); err == gorm.ErrRecordNotFound {
      panic("Could not find user in room.")
    }

    // delete user from room
    if tx != nil {
      tx.Delete(&roomUser)
      tx.Commit()
    } else if rs.RDB.Testmode {
      rs.RDB.DeleteRoomUser(&roomUser)
    }
  }()

  log.Printf(":LeaveRoom: Emitting resp=%v", resp)
  s.Emit("leaveRoomResp", resp, sioRoom)
}

//
//  Chatroom Handlers
//
func (rs *RoomzServer) NewChatroomMessageHandler(s socketio.Conn, data map[string]interface{}) {
  log.Printf(":NewChatroomMessage: Received data %v", data)

  var err error
  sioRoom := s.ID()
  respEvent := "incomingChatRoomMessage"
  resp := map[string]interface{}{
    "userID":    nil,
    "name":      nil,
    "message":   nil,
    "timestamp": nil,
    "success":   false,
  }

  // open fresh session
  tx := rs.RDB.Begin()
  func() {
    defer eventError(resp, tx)

    var roomId, userId, message string
    var ok bool
    roomId = fmt.Sprintf("%v", data["roomID"])
    if _, err = strconv.Atoi(roomId); err != nil {
      panic("Invalid room_id. Must be an integer.")
    }
    userId = fmt.Sprintf("%v", data["userID"])
    if _, err = strconv.Atoi(userId); err != nil {
      panic("Invalid user_id. Must be an integer.")
    }
    //username := fmt.Sprintf("%v", data["username"]) // UNUSED
    if message, ok = data["message"].(string); !ok {
      panic("Invalid message type.")
    }

    var room models.Room
    if room, err = rs.RDB.GetRoom(tx, roomId); err == gorm.ErrRecordNotFound {
      panic("Room does not exist!")
    }
  
    // direct chatroom message to correct socketio room
    sioRoom = roomId

    // retrieve user from room
    var roomUser models.RoomUser
    if roomUser, err = rs.RDB.GetRoomUser(tx, roomId, userId); err == gorm.ErrRecordNotFound {
      panic(fmt.Sprintf("Failed to find user with user_id=%v in room_id=%v", userId, roomId))
    }

    // create message object
    messageTimestamp := time.Now()
    chatroomMessage := models.Message{
      RoomId:        room.Id,
      UserId:        roomUser.UserId,
      Username:      roomUser.Username,
      Message:       message,
      Timestamp:     messageTimestamp,
      MessageTypeId: chatroomMessageTypeId,
    }
    if err = rs.RDB.CreateMessage(tx, &chatroomMessage); err != nil {
      panic(err.Error())
    }

    resp = map[string]interface{}{
      "userID":    roomUser.UserId,
      "name":      roomUser.Username,
      "message":   message,
      "timestamp": messageTimestamp.Format("3:04PM"),
      "success":   true,
    }
    if tx != nil {
      tx.Commit()
    }
  }()

  log.Printf(":NewChatroomMessage: Emitting response=%v to id=%v", resp, sioRoom)
  rs.Server.BroadcastToRoom("/", sioRoom, respEvent, resp)
}

//
// Strict Join Request Handlers
//
func (rs *RoomzServer) GetJoinRequestsHandler(s socketio.Conn, data map[string]interface{}) {
  log.Printf(":GetJoinRequests: Received data %v", data)

  var err error
  sioRoom := s.ID()
  respEvent := "getJoinRequestsResp"
  resp := map[string]interface{}{
    "success":       true,
    "room_id":       nil,
    "join_requests": nil,
  }

  // open fresh session
  tx := rs.RDB.Begin()
  func() {
    defer eventError(resp, tx)
    
    roomId := fmt.Sprintf("%v", data["room_id"])
    if _, err = strconv.Atoi(roomId); err != nil {
      panic("Invalid room_id. Must be an integer.")
    }
    // TODO: validate user_id as host of the room?
    // userId := fmt.Sprintf("%v", data["user_id"]) // UNUSED
  
    // retrieve room of focus
    var room models.Room
    if room, err = rs.RDB.GetRoom(tx, roomId); err == gorm.ErrRecordNotFound {
      panic(fmt.Sprintf("Failed to find room with room id: %v", roomId))
    }

    joinRequests := rs.RDB.GetRoomJoinRequests(tx, room.Id)
    resp["room_id"] = roomId
    resp["join_requests"] = joinRequests
    if tx != nil {
      tx.Commit()
    }
  }()

  log.Printf(":GetJoinRequests: Emitting response=%v", resp)
  s.Emit(respEvent, resp, sioRoom)
}


func (rs *RoomzServer) HandleJoinRequestHandler(s socketio.Conn, data map[string]interface{}) {
  log.Printf(":HandleJoinRequest: Received data %v", data)

  var err error
  sioRoom := s.ID()
  // prepare response
  respEvent := "joinRoomResp"
  joinRoomResp := map[string]interface{}{
    "success":       false,
    "room_id":       nil,
    "user_id":       nil,
    "status":        "reject",
    "chat_history":  nil,
    "error_message": nil,
  }

  // open fresh session
  tx := rs.RDB.Begin()
  var roomId string
  var room models.Room
  func() {
    defer eventError(joinRoomResp, tx)

    var userIdToHandle, decision string
    roomId = fmt.Sprintf("%v", data["room_id"])
    if _, err = strconv.Atoi(roomId); err != nil {
      panic("Invalid room_id. Must be an integer.")
    }
    userIdToHandle = fmt.Sprintf("%v", data["user_id_to_handle"])
    if _, err = strconv.Atoi(userIdToHandle); err != nil {
      panic("Invalid user_id_to_handle. Must be an integer.")
    }
    decision = fmt.Sprintf("%v", data["decision"])
    if decision != "accept" && decision != "reject" {
      panic("Invalid decision. Must be: [accept|reject]")
    }

    
    // retrieve room of focus
    if room, err = rs.RDB.GetRoom(tx, roomId); err == gorm.ErrRecordNotFound {
      panic(fmt.Sprintf("Failed to find room with room id: %v", roomId))
    }

    // retrieve roomJoinRequest
    var roomJoinRequest models.RoomJoinRequest
    if roomJoinRequest, err = rs.RDB.GetRoomJoinRequest(tx, roomId, userIdToHandle); err == gorm.ErrRecordNotFound {
      panic("No join request found!")
    }

    if acceptUser := decision == "accept"; acceptUser {
      // retrieve chatroom messages
      roomChatHistory := rs.RDB.GetRoomChatMessages(tx, roomId)
      log.Printf(":HandleJoinRequest: Retrieved chatroom history=%v", roomChatHistory)

      // retrieve user
      var user models.User
      if user, err = rs.RDB.GetUser(tx, userIdToHandle); err == gorm.ErrRecordNotFound {
        panic("User does not exist!")
      }

      // create new room user
      username := roomJoinRequest.Username
      newRoomUser := models.RoomUser{
        RoomId:   room.Id,
        UserId:   user.Id,
        Username: username,
      }

      rs.RDB.CreateRoomUser(tx, &newRoomUser)
      log.Printf(":HandleJoinRequest: Created new RoomUser=%v", newRoomUser)

      // create activity message
      activityMessageString := fmt.Sprintf("%v has joined the room", newRoomUser)
      activityMessage := models.Message{
        RoomId:        room.Id,
        UserId:        user.Id,
        Username:      username,
        Message:       activityMessageString,
        Timestamp:     time.Now(),
        MessageTypeId: activityMessageTypeId,
      }
      if err = rs.RDB.CreateMessage(tx, &activityMessage); err != nil {
        panic("Internal server error.")
      }
      log.Printf(":HandleJoinRequest: Created new AcitivityMessage=%v", activityMessage)

      joinRoomResp = map[string]interface{}{
        "success":       true,
        "room_id":       roomId,
        "user_id":       userIdToHandle,
        "status":        "accept",
        "chat_history":  roomChatHistory,
        "error_message": nil,
      }
    } else {
      // reject user
      joinRoomResp["success"] = true
      joinRoomResp["room_id"] = roomId
      joinRoomResp["user_id"] = userIdToHandle
      joinRoomResp["status"] = "reject"
    }
    // emit response to user trying to join
    sioRoom = roomJoinRequest.SessionId

    // delete remnant roomJoinRequest
    log.Printf(":HandleJoinRequest: Deleting RoomJoinRequest=%v", roomJoinRequest)
    if tx != nil {
      tx.Delete(&roomJoinRequest)
      tx.Commit()
    } else if rs.RDB.Testmode {
      rs.RDB.DeleteRoomJoinRequest(&roomJoinRequest)
    }
  }()

  // respond to user requesting to join if successful else error message to host
  log.Printf(":HandleJoinRequest: Emitting \"%v\" response=%v to sioRoom=%v", respEvent, joinRoomResp, sioRoom)
  rs.Server.BroadcastToRoom("/", sioRoom, respEvent, joinRoomResp)

  tx = rs.RDB.Begin()
  sioRoom = s.ID()
  if joinRoomResp["success"].(bool) {
    // send fresh emit to host, as join requests have been updated
    respEvent = "getJoinRequestsResp"
    joinRequestsRespData := map[string]interface{}{
      "success":       true,
      "room_id":       roomId,
      "join_requests": rs.RDB.GetRoomJoinRequests(tx, room.Id),
    }
    log.Printf(":HandleJoinRequest: Emitting \"%v\" response=%v", respEvent, joinRequestsRespData)
    s.Emit(respEvent, joinRequestsRespData, sioRoom)
  }

  // response event to host of handling the join request
  respEvent = "handleJoinRequestResp"
  resp := map[string]interface{}{
    "success":       joinRoomResp["success"],
    "error_message": joinRoomResp["error_message"],
  }
  s.Emit(respEvent, resp, sioRoom)
}

func (rs *RoomzServer) CancelJoinRequestHandler(s socketio.Conn, data map[string]interface{}) {
  log.Printf(":CancelJoinRequest: Received data %v", data)

  var err error
  sioRoom := s.ID()
  resp := map[string]interface{}{
    "success":       false,
  }

  // open fresh session
  tx := rs.RDB.Begin()
  func() {
    defer eventError(resp, tx)

    var roomId, userId string
    roomId = fmt.Sprintf("%v", data["room_id"])
    if _, ok := strconv.Atoi(roomId); ok != nil {
      panic("Invalid room_id, must be a number.")
    }
    userId = fmt.Sprintf("%v", data["user_id"])
    if _, ok := strconv.Atoi(userId); ok != nil {
      panic("Invalid user_id, must be a number.")
    }

    var room models.Room
    if room, err = rs.RDB.GetRoom(tx, roomId); err == gorm.ErrRecordNotFound {
      panic("Room does not exist!")
    }

    var roomJoinRequest models.RoomJoinRequest
    if roomJoinRequest, err = rs.RDB.GetRoomJoinRequest(tx, roomId, userId); err == gorm.ErrRecordNotFound {
      panic("No join request found!")
    }
    if tx != nil {
      tx.Delete(&roomJoinRequest)
    } else if rs.RDB.Testmode {
      rs.RDB.DeleteRoomJoinRequest(&roomJoinRequest)
    }
    log.Printf(":CancelJoinRequest: Deleted join request successfully.")

    // send notification to host
    sioRoom = fmt.Sprintf("%v", room.HostSid)
    resp["success"] = true
    if tx != nil {
      tx.Commit()
    }
  }()
  respEvent := "incomingJoinCancel"
  log.Printf(":CancelJoinRequest: Emitting \"%v\" with resp=%v", respEvent, resp)
  rs.Server.BroadcastToRoom("/", sioRoom, respEvent, resp)

  // deliver receipt to canceller
  respEvent = "cancelJoinRequestResp"
  sioRoom = s.ID()
  log.Printf(":CancelJoinRequest: Emitting \"%v\" with resp=%v", respEvent, resp)
  s.Emit(respEvent, resp, sioRoom)
}

//
// Refresh Connection
//
func (rs *RoomzServer) UpdateSessionIdHandler(s socketio.Conn, data map[string]interface{}) {
  log.Printf(":UpdateSessionId: Received data %v", data)

  var err error
  sioRoom := s.ID()
  respEvent := "updateSessionIdResp"
  resp := map[string]interface{}{
    "success":       false,
  }

  // open fresh session
  tx := rs.RDB.Begin()
  func() {
    defer eventError(resp, tx)
    var roomId string
    roomId = fmt.Sprintf("%v", data["room_id"])
    if _, ok := strconv.Atoi(roomId); ok != nil {
      panic("Invalid room_id. Must be a number.")
    }

    // if room exists, re-join room
    if _, err = rs.RDB.GetRoom(tx, roomId); err == gorm.ErrRecordNotFound {
      panic("Room does not exist!")
    }

    // TODO: edit FE to send is_host for a host session ID to be updated.
    log.Printf(":UpdateSessionId: Session Id=%v has re-joined room=%v", s.ID(), roomId)
    rs.Server.JoinRoom("/", roomId, s)

    // check if user has any pending join requests
    if userIdTmp, ok := data["user_id"]; ok {
      userId := fmt.Sprintf("%v", userIdTmp)
      if _, ok := strconv.Atoi(userId); ok != nil {
        panic("Invalid user_id. Must be a number.")
      }

      var roomJoinRequest models.RoomJoinRequest
      if roomJoinRequest, err = rs.RDB.GetRoomJoinRequest(tx, roomId, userId); err == gorm.ErrRecordNotFound {
        panic("No join request found!")
      }

      // update join request with latest session id
      roomJoinRequest.SessionId = s.ID()
      if tx != nil {
        tx.Save(&roomJoinRequest)
      }
      log.Printf(":UpdateSessionId: Updated latest session id in join request")
    }
    resp["success"] = true
    if tx != nil {
      tx.Commit()
    }
  }()

  log.Printf(":UpdateSessionId: Emitting \"%v\" response=%v", respEvent, resp)
  s.Emit(respEvent, resp, sioRoom)
}
*/