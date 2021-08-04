// The server package implements a gRPC RoomzApiService server.
package server

import (
  "flag"
  "fmt"
  "sync"
  "os"

  rpb "github.com/ABFranco/roomz-proto/go_proto"
  "github.com/ABFranco/roomz-api-server/migration"
  "github.com/ABFranco/roomz-api-server/models"
  "google.golang.org/grpc"
  "gorm.io/driver/postgres"
  "gorm.io/gorm"
)

const (
  host     = "localhost"
  port     = 5432
  dbname   = "roomz"
)

var (
  testMode = flag.Bool("test", false, "indicate running the RAS in test mode.")
  dbInit = flag.Bool("init", false, "initialize the Postgres DB")
  dbMigrate = flag.Bool("migrate", false, "migrate the Postgres DB")
)

type roomUserChatChannel struct {
  userId uint32
  channel chan *rpb.ChatMessage
}
type roomUserCloseChannel struct {
  userId uint32
  channel chan *rpb.HostClosedRoom
} 

type roomzApiService struct {
  RDB             *models.RoomzDB

  tknsMtx         sync.RWMutex
  ActiveTkns      []string

  joinStreamsMtx  sync.RWMutex
  RoomJoinStreams map[string]chan *rpb.JoinRoomResponse
  
  chatStreamsMtx  sync.RWMutex
  RoomChatStreams map[uint32][]roomUserChatChannel

  closeStreamsMtx  sync.RWMutex
  RoomCloseStreams map[uint32][]roomUserCloseChannel
}

func NewRoomzApiServer() *grpc.Server {
  // Initialize the database.
  var err error
  user     := os.Getenv("ROOMZ_DB_USER")
  password := os.Getenv("ROOMZ_DB_PASSWORD")

  psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
    host, port, user, password, dbname)
  
  db, err := gorm.Open(postgres.Open(psqlInfo), &gorm.Config{})
  if err != nil {
    panic(err)
  }

  flag.Parse()
  if *dbInit {
    migration.Init(db)
    return nil
  }
  if *dbMigrate {
    migration.Migrate(db)
    return nil
  }
  if *testMode {
    fmt.Printf("Running test mode!!\n")
  }
  roomzDB := &models.RoomzDB{
    DB:       db,
    Testmode: *testMode,
  }

  // Initialize the gRPC service.
  server := grpc.NewServer()
  rpb.RegisterRoomzApiServiceServer(server, &roomzApiService{
    RDB: roomzDB,
    RoomChatStreams:  make(map[uint32][]roomUserChatChannel),
    RoomCloseStreams: make(map[uint32][]roomUserCloseChannel),
    RoomJoinStreams:  make(map[string]chan *rpb.JoinRoomResponse),
  })
  return server
}
