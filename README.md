# plz.service

concrete implementation choices for plz service

# http

server

```go
func sayHello(ctx *countlog.Context, req *MyReqeust) (*MyResponse, error) {
	// ...
}
server := http.NewServer()
server.Handle("/sayHello", sayHello)
server.Start("127.0.0.1:9998")
```

client

```go
var sayHello = func (ctx *countlog.Context, req *MyReqeust) (*MyResponse, error)
client := http.NewClient()
client.Handle("POST", "http://127.0.0.1:9998/sayHello", &sayHello)

// use sayHello(...) to call server
```

# thrift

server 

```go
func sayHello(ctx *countlog.Context, req *MyReqeust) (*MyResponse, error) {
	// ...
}
server := thrift.NewServer()
server.Handle("sayHello", sayHello)
server.Start("127.0.0.1:9998")
```

client

```go
var sayHello = func (ctx *countlog.Context, req *MyReqeust) (*MyResponse, error)
client := thrift.NewClient()
client.Handle("127.0.0.1:9998", "sayHello", &sayHello)

// use sayHello(...) to call server
```