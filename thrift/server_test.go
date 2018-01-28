package thrift

import (
	"testing"
	"github.com/v2pro/plz/countlog"
	"context"
	"github.com/stretchr/testify/require"
	"time"
	"runtime/debug"
	"github.com/thrift-iterator/go"
)

func Test_normal_response(t *testing.T) {
	debug.SetGCPercent(-1)
	should := require.New(t)
	type TestRequest struct {
		Field1 string `thrift:",1"`
	}
	type TestResponse struct {
		Field2 string `thrift:",1"`
	}
	//server := NewServer(thrifter.Config{Protocol: thrifter.ProtocolBinary, IsFramed: true}.Froze())
	server := NewServer(thrifter.DefaultConfig)
	server.Handle("sayHello", func(ctx *countlog.Context, req *TestRequest) (*TestResponse, error) {
		return &TestResponse{
			Field2: "hello",
		}, nil
	})
	go server.Start("127.0.0.1:9998")
	time.Sleep(time.Millisecond * 100)
	defer server.Stop()
	//client := NewClient(thrifter.Config{Protocol: thrifter.ProtocolBinary, IsFramed: true}.Froze())
	client := NewClient(thrifter.DefaultConfig)
	var sayHello func(ctx *countlog.Context, req *TestRequest) (*TestResponse, error)
	client.Handle("127.0.0.1:9998", "sayHello", &sayHello)

	ctx := countlog.Ctx(context.Background())
	resp, err := sayHello(ctx, &TestRequest{"hello"})
	should.NoError(err)
	should.Equal("hello", resp.Field2)
}
