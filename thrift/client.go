package thrift

import (
	"github.com/thrift-iterator/go"
	"github.com/thrift-iterator/go/protocol"
	"github.com/v2pro/plz/countlog"
	"github.com/v2pro/plz/service"
	"net"
	"time"
	"unsafe"
)

type Client struct {
	thriftApi thrifter.API
}

// TODO: support framed transport
func NewClient(thriftApi thrifter.API) *Client {
	return &Client{
		thriftApi: thriftApi,
	}
}

func (client *Client) Handle(addr string, messageName string, ptrHandlerObj interface{}) {
	ptrHandler, handlerTypeInfo := service.ConvertPtrHandler(ptrHandlerObj)
	*ptrHandler = func(ctx *countlog.Context, ptrReq unsafe.Pointer) (unsafe.Pointer, error) {
		reqObj := handlerTypeInfo.RequestBoxer(ptrReq)
		// TODO: connection pool
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err != nil {
			return nil, err
		}
		encoder := client.thriftApi.NewEncoder(conn)
		conn.SetWriteDeadline(time.Now().Add(time.Second))
		err = encoder.EncodeMessageHeader(protocol.MessageHeader{
			MessageName: messageName,
			MessageType: protocol.MessageTypeCall,
		})
		if err != nil {
			return nil, err
		}
		conn.SetWriteDeadline(time.Now().Add(time.Second))
		err = encoder.Encode(reqObj)
		if err != nil {
			return nil, err
		}
		time.Sleep(time.Second)
		decoder := client.thriftApi.NewDecoder(conn, nil)
		conn.SetReadDeadline(time.Now().Add(time.Second))
		_, err = decoder.DecodeMessageHeader()
		if err != nil {
			return nil, err
		}
		var resp unsafe.Pointer
		ptrResp := unsafe.Pointer(&resp)
		respObj := handlerTypeInfo.ResponseBoxer(ptrResp)
		conn.SetReadDeadline(time.Now().Add(time.Second))
		err = decoder.Decode(respObj)
		if err != nil {
			return nil, err
		}
		return ptrResp, nil
	}
}
