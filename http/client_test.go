package http

import (
	"testing"
	"net/http"
	"github.com/v2pro/plz/countlog"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"context"
	"time"
)

func Test_client(t *testing.T) {
	should := require.New(t)
	type TestRequest struct {
		Field string
	}
	type TestResponse struct {
		Field string
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/sayHello", func(writer http.ResponseWriter, request *http.Request) {
		reqBody, err := ioutil.ReadAll(request.Body)
		should.NoError(err)
		should.Equal(`{"Field":"hello"}`, string(reqBody))
		writer.Write([]byte(`{"Field":"world"}`))
	})
	go http.ListenAndServe("127.0.0.1:9997", mux)
	time.Sleep(time.Millisecond * 100)

	client := NewClient(http.DefaultClient)
	var sayHello func(*countlog.Context, *TestRequest)(*TestResponse, error)
	client.Handle("POST", "http://127.0.0.1:9997/sayHello", &sayHello)

	ctx := countlog.Ctx(context.Background())
	resp, err := sayHello(ctx, &TestRequest{"hello"})
	should.NoError(err)
	should.Equal("world", resp.Field)
}
