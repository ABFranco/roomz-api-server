// Package migration manages database initializations or migrations.
package migration

import (
  "fmt"

  "github.com/ABFranco/roomz-api-server/models"
  "gorm.io/gorm"
)

// Init checks for the existence of all expected database tables, and if non-existant,
// automatically creates them.
func Init(db *gorm.DB) {
  migrator := db.Migrator()
  fmt.Printf("Initializing the db: %s...\n", migrator.CurrentDatabase())

  // Verify the existence of all models & create if not already existed.
  if !migrator.HasTable(&models.Guest{}) {
    fmt.Println("The database does not have a guest table. Creating...")
    if err := migrator.CreateTable(&models.Guest{}); err != nil {
      fmt.Println("Failed to create guest table...")
      return
    }
  }
  if !migrator.HasTable(&models.Account{}) {
    fmt.Println("The database does not have a account table. Creating...")
    if err := migrator.CreateTable(&models.Account{}); err != nil {
      fmt.Println("Failed to create account table...")
      return
    }
  }
  if !migrator.HasTable(&models.RoomUser{}) {
    fmt.Println("The database does not have a room user table. Creating...")
    if err := migrator.CreateTable(&models.RoomUser{}); err != nil {
      fmt.Println("Failed to create room user table...")
      return
    }
  }
  if !migrator.HasTable(&models.User{}) {
    fmt.Println("The database does not have a user table. Creating...")
    if err := migrator.CreateTable(&models.User{}); err != nil {
      fmt.Println("Failed to create user table...")
      return
    }
  }
  if !migrator.HasTable(&models.Message{}) {
    fmt.Println("The database does not have a message table. Creating...")
    if err := migrator.CreateTable(&models.Message{}); err != nil {
      fmt.Println("Failed to create message table...")
      return
    }
  }
  if !migrator.HasTable(&models.MessageType{}) {
    fmt.Println("The database does not have a message type table. Creating...")
    if err := migrator.CreateTable(&models.MessageType{}); err != nil {
      fmt.Println("Failed to create message type table...")
      return
    }
    // Initialize 2 rows in MessageType table.
    tx := db.Begin()
    chatroomMessageType := models.MessageType{
      MessageType: "chatroom",
    }
    if tx.Create(&chatroomMessageType).Error != nil {
      fmt.Println("Failed to create chatroom message type!")
      return
    }
    activityMessageType := models.MessageType{
      MessageType: "activity",
    }
    if tx.Create(&activityMessageType).Error != nil {
      fmt.Println("Failed to create activity message type!")
      return
    }
    tx.Commit()
  }
  if !migrator.HasTable(&models.RoomJoinRequest{}) {
    fmt.Println("The database does not have a room join request table. Creating...")
    if err := migrator.CreateTable(&models.RoomJoinRequest{}); err != nil {
      fmt.Println("Failed to create room join request table...")
      return
    }
  }
  if !migrator.HasTable(&models.Room{}) {
    fmt.Println("The database does not have a room table. Creating...")
    if err := migrator.CreateTable(&models.Room{}); err != nil {
      fmt.Println("Failed to create room table...")
      return
    }
  }
  fmt.Printf("Database %s successfully initialized!", migrator.CurrentDatabase())
}

// Migrate automatically migrates the Postgres DB for any schema changes (i.e. new structs or 
// altered structs in models.go).
func Migrate(db *gorm.DB) {
  migrator := db.Migrator()
  fmt.Printf("Auto-Migrating db %s for any schema changes...\n", migrator.CurrentDatabase())
  fmt.Println("NOTE: Auto-migration will not automatically delete columns.")
  
  if err := migrator.AutoMigrate(
    &models.Guest{},
    &models.Account{},
    &models.RoomUser{},
    &models.User{},
    &models.Message{},
    &models.MessageType{},
    &models.RoomJoinRequest{},
    &models.Room{},
  ); err != nil {
    fmt.Printf("Error auto-migrating db  %s: %v", migrator.CurrentDatabase(), err)
    return
  }
  fmt.Printf("Database %s successfully migrated!", migrator.CurrentDatabase())
}