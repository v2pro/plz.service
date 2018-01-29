package http

import (
	"bytes"
	"errors"
	"github.com/stretchr/testify/require"
	"github.com/v2pro/plz/countlog"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

func init() {
	countlog.MinLevel = countlog.LevelTrace
}

func Test_should_panic_if_type_not_matching_handler_prototype(t *testing.T) {
	should := require.New(t)
	server := NewServer()
	should.Panics(func() {
		server.Handle("/", func(ctx *countlog.Context, req int) (*int, error) {
			return nil, nil
		})
	})
}

func Test_normal_response(t *testing.T) {
	should := require.New(t)
	server := NewServer()
	type TestRequest struct {
		Field1 string
	}
	type TestResponse struct {
		Field2 string
	}
	server.Handle("/", func(ctx *countlog.Context, req *TestRequest) (*TestResponse, error) {
		return &TestResponse{
			Field2: req.Field1,
		}, nil
	})
	go server.Start("127.0.0.1:9998")
	defer server.Close()
	time.Sleep(time.Millisecond * 100)
	resp, err := httpPost("http://127.0.0.1:9998", "application/json",
		bytes.NewBufferString(`{"Field1":"hello"}`))
	should.NoError(err)
	body, err := ioutil.ReadAll(resp.Body)
	should.NoError(err)
	should.Equal(`{"errno":0,"data":{"Field2":"hello"}}`, string(body))
}

func Test_error_response(t *testing.T) {
	should := require.New(t)
	server := NewServer()
	type TestRequest struct {
		Field1 string
	}
	type TestResponse struct {
		Field2 string
	}
	server.Handle("/", func(ctx *countlog.Context, req *TestRequest) (*TestResponse, error) {
		return nil, errors.New("fake error")
	})
	go server.Start("127.0.0.1:9998")
	defer server.Close()
	time.Sleep(time.Millisecond * 100)
	resp, err := httpPost("http://127.0.0.1:9998", "application/json",
		bytes.NewBufferString(`{}`))
	should.NoError(err)
	body, err := ioutil.ReadAll(resp.Body)
	should.NoError(err)
	should.Equal(`{"errno":1,"errmsg":"fake error","data":null}`, string(body))
}

type MyError struct {
}

func (err *MyError) Error() string {
	return "my error"
}

func (err *MyError) ErrorNumber() int {
	return 1024
}

func Test_error_number(t *testing.T) {
	should := require.New(t)
	server := NewServer()
	type TestRequest struct {
		Field1 string
	}
	type TestResponse struct {
		Field2 string
	}
	server.Handle("/", func(ctx *countlog.Context, req *TestRequest) (*TestResponse, error) {
		return nil, &MyError{}
	})
	go server.Start("127.0.0.1:9998")
	defer server.Close()
	time.Sleep(time.Millisecond * 100)
	resp, err := httpPost("http://127.0.0.1:9998", "application/json",
		bytes.NewBufferString(`{}`))
	should.NoError(err)
	body, err := ioutil.ReadAll(resp.Body)
	should.NoError(err)
	should.Equal(`{"errno":1024,"errmsg":"my error","data":null}`, string(body))
}

func httpPost(url string, contentType string, body io.Reader) (resp *http.Response, err error) {
	client := http.Client{
		Transport: &http.Transport{},
	}
	return client.Post(url, contentType, body)
}
