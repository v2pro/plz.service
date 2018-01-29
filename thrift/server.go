package thrift

import (
	"github.com/thrift-iterator/go"
	"github.com/thrift-iterator/go/protocol"
	"github.com/v2pro/plz/concurrent"
	"github.com/v2pro/plz/countlog"
	"github.com/v2pro/plz/service"
	"net"
	"time"
	"unsafe"
)

type Server struct {
	executor  *concurrent.UnboundedExecutor
	handlers  map[string]*thriftHandler
	thriftApi thrifter.API
}

// TODO: support framed transport
func NewServer(thriftApi thrifter.API) *Server {
	return &Server{
		executor:  concurrent.NewUnboundedExecutor(),
		handlers:  map[string]*thriftHandler{},
		thriftApi: thriftApi,
	}
}

func (server *Server) Handle(messageName string, handlerObj interface{}) {
	handler, handlerTypeInfo := service.ConvertHandler(handlerObj)
	server.handlers[messageName] = &thriftHandler{
		handler:         handler,
		handlerTypeInfo: handlerTypeInfo,
	}
}

func (server *Server) Start(addr string) error {
	listener, err := net.Listen("tcp", addr)
	countlog.TraceCall("callee!net.Listen", err)
	if err != nil {
		return err
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		server.executor.Go(func(ctx *countlog.Context) {
			server.handleConn(ctx, conn)
		})
		return nil
	}
}

func (server *Server) handleConn(ctx *countlog.Context, conn net.Conn) {
	defer conn.Close()
	// TODO: keep conn alive
	decoder := server.thriftApi.NewDecoder(conn, nil)
	conn.SetReadDeadline(time.Now().Add(time.Second))
	header, err := decoder.DecodeMessageHeader()
	ctx.TraceCall("callee!decoder.DecodeMessageHeader", err)
	if err != nil {
		return
	}
	handler := server.handlers[header.MessageName]
	if handler == nil {
		ThriftException{"handler not defined for this message"}.reply(ctx, header.MessageName, conn, server.thriftApi)
		return
	}
	var req unsafe.Pointer
	ptrReq := unsafe.Pointer(&req)
	reqObj := handler.handlerTypeInfo.RequestBoxer(ptrReq)
	err = decoder.Decode(reqObj)
	if err != nil {
		ThriftException{err.Error()}.reply(ctx, header.MessageName, conn, server.thriftApi)
		return
	}
	resp, err := handler.handler(ctx, ptrReq)
	if err != nil {
		ThriftException{err.Error()}.reply(ctx, header.MessageName, conn, server.thriftApi)
		return
	}
	respObj := handler.handlerTypeInfo.ResponseBoxer(resp)
	encoder := server.thriftApi.NewEncoder(conn)
	conn.SetWriteDeadline(time.Now().Add(time.Second))
	err = encoder.EncodeMessageHeader(protocol.MessageHeader{
		MessageName: header.MessageName,
		MessageType: protocol.MessageTypeReply,
	})
	ctx.TraceCall("callee!encoder.EncodeMessageHeader", err)
	if err != nil {
		return
	}
	conn.SetWriteDeadline(time.Now().Add(time.Second))
	err = encoder.Encode(respObj)
	ctx.TraceCall("callee!encoder.Encode", err)
}

func (server *Server) Stop() error {
	return nil
}

func (server *Server) Shutdown(ctx *countlog.Context) error {
	return server.Stop()
}

type thriftHandler struct {
	handler         service.Handler
	handlerTypeInfo *service.HandlerTypeInfo
}

type ThriftException struct {
	ErrorMessage string `thrift:",1"`
}

func (exception ThriftException) reply(ctx *countlog.Context, messageName string, conn net.Conn, thriftApi thrifter.API) {
	encoder := thriftApi.NewEncoder(conn)
	conn.SetWriteDeadline(time.Now().Add(time.Second))
	err := encoder.EncodeMessageHeader(protocol.MessageHeader{
		MessageName: messageName,
		MessageType: protocol.MessageTypeException,
	})
	ctx.TraceCall("callee!encoder.EncodeMessageHeader", err)
	if err != nil {
		return
	}
	conn.SetWriteDeadline(time.Now().Add(time.Second))
	err = encoder.Encode(exception)
	ctx.TraceCall("callee!encoder.Encode", err)
}
