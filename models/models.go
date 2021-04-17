package models

import (
  "database/sql"
  "fmt"
  "strconv"
  "time"

  rpb "github.com/ABFranco/roomz-proto/go_proto"
  "github.com/ABFranco/roomz-api-server/util"
  "gorm.io/gorm"
)

type Guest struct {
  Id     int
  UserId int
}

type Account struct {
  Id        int
  UserId    int
  Email     string
  Password  string
  FirstName string
  LastName  string
  StartDate time.Time
}

type RoomUser struct {
  Id       int
  RoomId   int
  UserId   int
  Username string
  JoinDate time.Time
}

type User struct {
  Id   int
  Name sql.NullString
}

type Message struct {
  Id            int
  RoomId        int
  UserId        int
  Username      string
  Message       string
  Timestamp     time.Time
  MessageTypeId int
}

type MessageType struct {
  Id          int
  MessageType string
}

type RoomJoinRequest struct {
  Id        int
  RoomId    int
  UserId    int
  Username  string
  SessionId string
  Expired   bool
}

type Room struct {
  Id           int
  HostId       int
  HostSid      string
  Password     string
  CreationDate time.Time
  IsStrict     bool
  UserLimit    int
  IsActive     bool
}

type mockDB struct {
  Guests           []Guest
  Accounts         []Account
  RoomUsers        []RoomUser
  Users            []User
  Messages         []Message
  RoomJoinRequests []RoomJoinRequest
  Rooms            []Room
}

type RoomzDB struct {
  DB       *gorm.DB
  Testmode bool
  mdb      mockDB
}

//
// Overload common functions
//
func (rdb *RoomzDB) Begin() *gorm.DB {
  if rdb.Testmode {
    return nil
  }
  return rdb.DB.Begin()
}

func (rdb *RoomzDB) Commit() *gorm.DB {
  if rdb.Testmode {
    return nil
  }
  return rdb.DB.Commit()
}

func (rdb *RoomzDB) Rollback() *gorm.DB {
  if rdb.Testmode {
    return nil
  }
  return rdb.DB.Rollback()
}

//
// Guest
//
func (rdb *RoomzDB) CreateGuest(tx *gorm.DB, guest *Guest) (error) {
  if rdb.Testmode {
    guest.Id = len(rdb.mdb.Guests)
    rdb.mdb.Guests = append(rdb.mdb.Guests, *guest)
    return nil
  }
  return tx.Create(guest).Error
}


//
// Account
//
func (rdb *RoomzDB) CreateAccount(tx *gorm.DB, account *Account) (error) {
  if rdb.Testmode {
    account.Id = len(rdb.mdb.Accounts)
    rdb.mdb.Accounts = append(rdb.mdb.Accounts, *account)
    return nil
  }
  return tx.Create(account).Error
}

func (rdb *RoomzDB) GetAccount(tx *gorm.DB, email string) (Account, error) {
  var account Account
  if rdb.Testmode {
    for _, act := range rdb.mdb.Accounts {
      if act.Email == email {
        return act, nil
      }
    }
    return account, gorm.ErrRecordNotFound
  }
  err := tx.Where("email = ?", email).First(&account).Error
  return account, err
}

//
// Account Details 
//
func (rdb *RoomzDB) EditAccountEmail(tx *gorm.DB, old_email string, new_email string) (error) {
  if rdb.Testmode {
    for _, act := range rdb.mdb.Accounts {
      if act.Email == old_email {
        act.Email = new_email
        return nil
      }
    } 
    return gorm.ErrRecordNotFound
  }
  var account Account 
  err := tx.Where("email = ?", old_email).First(&account).Error
  if err == gorm.ErrRecordNotFound {
    return err
  }
  account.Email = new_email
  tx.Save(&account)
  tx.Commit()
  return err
}

func (rdb *RoomzDB) EditAccountPassword(tx *gorm.DB, email string, new_password string) (error) {
  if rdb.Testmode {
    for _, act := range rdb.mdb.Accounts {
      if act.Email == email {
        act.Password = util.Encrypt(new_password)
        return nil
      }
    }
    return gorm.ErrRecordNotFound
  }
  var account Account 
  err := tx.Where("email = ?", email).First(&account).Error
  if err == gorm.ErrRecordNotFound {
    return err
  } 
  account.Password = util.Encrypt(new_password)
  tx.Save(&account)
  tx.Commit()
  return err 
}

//
// User
//
func (rdb *RoomzDB) CreateUser(tx *gorm.DB, user *User) (error) {
  if rdb.Testmode {
    user.Id = len(rdb.mdb.Users)
    rdb.mdb.Users = append(rdb.mdb.Users, *user)
    return nil
  }
  return tx.Create(user).Error
}

func (rdb *RoomzDB) GetUser(tx *gorm.DB, userId string) (User, error) {
  var user User
  if rdb.Testmode {
    // assume userId is a valid integer (error checking done before)
    userIdInt, _ := strconv.Atoi(userId)
    for _, usr := range rdb.mdb.Users {
      if usr.Id == userIdInt {
        return usr, nil
      }
    }
    return user, gorm.ErrRecordNotFound
  }
  err := tx.Where("id = ?", userId).First(&user).Error
  return user, err
}


//
// RoomUser
//
func (rdb *RoomzDB) CreateRoomUser(tx *gorm.DB, roomUser *RoomUser) (error) {
  if rdb.Testmode {
    roomUser.Id = len(rdb.mdb.RoomUsers)
    rdb.mdb.RoomUsers = append(rdb.mdb.RoomUsers, *roomUser)
    return nil
  }
  return tx.Create(roomUser).Error
}

func (rdb *RoomzDB) GetRoomUser(tx *gorm.DB, roomId, userId string) (RoomUser, error) {
  var roomUser RoomUser
  if rdb.Testmode {
    // assume roomId and userId is a valid integer (error checking done before)
    roomIdInt, _ := strconv.Atoi(roomId)
    userIdInt, _ := strconv.Atoi(userId)
    
    for _, roomUsr := range rdb.mdb.RoomUsers {
      if roomUsr.UserId == userIdInt && roomUsr.RoomId == roomIdInt {
        return roomUsr, nil
      }
    }
    return roomUser, gorm.ErrRecordNotFound
  }
  err := tx.Where("room_id = ? AND user_id = ?", roomId, userId).First(&roomUser).Error
  return roomUser, err
}

func (rdb *RoomzDB) DeleteRoomUser(roomUser *RoomUser) {
  for i, roomUsr := range rdb.mdb.RoomUsers {
    if roomUsr.UserId == roomUser.UserId && roomUsr.RoomId == roomUser.RoomId {
      rdb.mdb.RoomUsers = append(rdb.mdb.RoomUsers[:i], rdb.mdb.RoomUsers[i+1:]...)
    }
  }
}


//
// Room
//
func (rdb *RoomzDB) CreateRoom(tx *gorm.DB, room *Room) (error) {
  if rdb.Testmode {
    room.Id = len(rdb.mdb.Rooms)
    rdb.mdb.Rooms = append(rdb.mdb.Rooms, *room)
    return nil
  }
  return tx.Create(room).Error
}

func (rdb *RoomzDB) GetRoom(tx *gorm.DB, roomId string) (Room, error) {
  var room Room
  if rdb.Testmode {
    // assume roomId is a valid integer (error checking done before)
    roomIdInt, _ := strconv.Atoi(roomId)
    for _, rm := range rdb.mdb.Rooms {
      if rm.Id == roomIdInt {
        return rm, nil
      }
    }
    return room, gorm.ErrRecordNotFound
  }
  err := tx.Where("id = ?", roomId).First(&room).Error
  return room, err
}


//
// RoomJoinRequest
//
func (rdb *RoomzDB) CreateRoomJoinRequest(tx *gorm.DB, roomJoinRequest *RoomJoinRequest) (error) {
  if rdb.Testmode {
    roomJoinRequest.Id = len(rdb.mdb.RoomJoinRequests)
    rdb.mdb.RoomJoinRequests = append(rdb.mdb.RoomJoinRequests, *roomJoinRequest)
    return nil
  }
  return tx.Create(roomJoinRequest).Error
}

func (rdb *RoomzDB) GetRoomJoinRequest(tx *gorm.DB, roomId string, userId string) (RoomJoinRequest, error) {
  var roomJoinRequest RoomJoinRequest
  if rdb.Testmode {
    // assume roomId and userId is a valid integer (error checking done before)
    roomIdInt, _ := strconv.Atoi(roomId)
    userIdInt, _ := strconv.Atoi(userId)
    
    for _, rjr := range rdb.mdb.RoomJoinRequests {
      if rjr.UserId == userIdInt && rjr.RoomId == roomIdInt {
        return rjr, nil
      }
    }
    return roomJoinRequest, gorm.ErrRecordNotFound
  }
  err := tx.Where("room_id = ? AND user_id = ?", roomId, userId).First(&roomJoinRequest).Error
  return roomJoinRequest, err
}

func (rdb *RoomzDB) GetRoomJoinRequests(tx *gorm.DB, roomId int) ([]*rpb.StrictJoinRequest) {
  joinRequests := []*rpb.StrictJoinRequest{}
  var roomJoinRequests []RoomJoinRequest
  if rdb.Testmode {
    for _, rjr := range rdb.mdb.RoomJoinRequests {
      if rjr.RoomId == roomId {
        joinRequest := &rpb.StrictJoinRequest{
          RoomId:   uint32(roomId),
          UserId:   uint32(rjr.UserId),
          UserName: rjr.Username,
        }
        joinRequests = append(joinRequests, joinRequest)
      }
    }
    return joinRequests
  }

  tx.Where("room_id = ?", roomId).Find(&roomJoinRequests)
  for _, roomJoinRequest := range roomJoinRequests {
    joinRequest := &rpb.StrictJoinRequest{
      RoomId: uint32(roomId),
      UserId: uint32(roomJoinRequest.UserId),
      UserName: roomJoinRequest.Username,
    }
    joinRequests = append(joinRequests, joinRequest)
  }

  return joinRequests
}

func (rdb *RoomzDB) DeleteRoomJoinRequest(roomJoinRequest *RoomJoinRequest) {
  for i, rjr := range rdb.mdb.RoomJoinRequests {
    if rjr.UserId == roomJoinRequest.UserId && rjr.RoomId == roomJoinRequest.RoomId {
      rdb.mdb.RoomJoinRequests = append(rdb.mdb.RoomJoinRequests[:i], rdb.mdb.RoomJoinRequests[i+1:]...)
    }
  }
}

//
// Message
//
func (rdb *RoomzDB) CreateMessage(tx *gorm.DB, message *Message) (error) {
  if rdb.Testmode {
    message.Id = len(rdb.mdb.Messages)
    rdb.mdb.Messages = append(rdb.mdb.Messages, *message)
    return nil
  }
  return tx.Create(message).Error
}

func (rdb *RoomzDB) GetRoomChatMessages(tx *gorm.DB, roomId string) ([]*rpb.ChatMessage) {
  roomChatHistory := []*rpb.ChatMessage{}
  var messages []Message
  if rdb.Testmode {
    // assume roomId is a valid integer (error checking done before)
    roomIdInt, _ := strconv.Atoi(roomId)

    for _, m := range rdb.mdb.Messages {
      if m.RoomId == roomIdInt && m.MessageTypeId == 1 {
          roomChatHistory = append(roomChatHistory, &rpb.ChatMessage{
          RoomId: uint32(m.RoomId),
          UserId: uint32(m.UserId),
          Message: m.Message,
          UserName: m.Username,
          Timestamp: fmt.Sprintf("%v", m.Timestamp.Format("3:04PM")),
        })
      }
    }
    return roomChatHistory
  }

  // TODO: define enums
  tx.Where("room_id = ? AND message_type_id = ?", roomId, 1).Find(&messages)
  for _, m := range messages {
    roomChatHistory = append(roomChatHistory, &rpb.ChatMessage{
      RoomId: uint32(m.RoomId),
      UserId: uint32(m.UserId),
      Message: m.Message,
      UserName: m.Username,
      Timestamp: fmt.Sprintf("%v", m.Timestamp.Format("3:04PM")),
    })
  }
  return roomChatHistory
}