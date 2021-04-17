# Roomz API Server
A gRPC Roomz API server which uses a PostgreSQL DB to interface with user data.

## Requirements
- Golang (version 1.12+)

## Normal Ops
Start the server:
```
$ go run main.go
```

Currently running on port 9090.

## Testing
1. Start the server on another terminal window.

Add the commandline flag `-test` to run in test mode where we use mocks in place of the true PostgreSQL DB as to not dirty it.
```
go run main.go -test
```

2. Run the tests
```
cd test
go test -v
```

## Potential Issues
If the server does not shut down correctly, it could still be running on port 5000 and you'll need to kill that process. Add this to your bashrc:
```
close_port() {
  kill -9 $(lsof -t -i:"$1")
}
```
You can then do `close_port 9090` if you run into issues.