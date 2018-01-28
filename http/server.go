package http

import (
	"net/http"
	"github.com/v2pro/plz/countlog"
	"io/ioutil"
	"unsafe"
	"github.com/v2pro/plz/plzio"
	"net"
	"time"
)

type Server struct {
	Unmarshaller plzio.Unmarshaller
	Marshaller   plzio.Marshaller
	mux          *http.ServeMux
	listener     net.Listener
}

func NewServer() *Server {
	mux := &http.ServeMux{}
	return &Server{
		Unmarshaller: &httpServerUnmarshaller{plzio.NewJsoniterUnmarshaller()},
		Marshaller:   &httpServerMarshaller{plzio.NewJsoniterResponseMarshaller()},
		mux:          mux,
	}
}

func (server *Server) Start(addr string) error {
	srv := &http.Server{Addr: addr, Handler: server.mux}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		countlog.Error("event!failed to listen http", "err", err)
		return err
	}
	server.listener = ln
	return srv.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)})
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

func (server *Server) Stop() error {
	if server.listener == nil {
		return nil
	}
	return server.listener.Close()
}

func (server *Server) Handle(pattern string, handlerObj interface{}) {
	handler, handlerTypeInfo := plzio.ConvertHandler(handlerObj)
	server.mux.Handle(pattern, &handlerAdapter{
		marshaller:      server.Marshaller,
		unmarshaller:    server.Unmarshaller,
		handler:         handler,
		handlerTypeInfo: handlerTypeInfo,
	})
}

type handlerAdapter struct {
	unmarshaller    plzio.Unmarshaller
	marshaller      plzio.Marshaller
	handler         plzio.Handler
	handlerTypeInfo *plzio.HandlerTypeInfo
}

func (adapter *handlerAdapter) ServeHTTP(httpRespWriter http.ResponseWriter, httpReq *http.Request) {
	ctxObj := httpReq.Context()
	ctx := countlog.Ctx(ctxObj)
	var req unsafe.Pointer
	ptrReq := unsafe.Pointer(&req)
	reqObj := adapter.handlerTypeInfo.RequestBoxer(ptrReq)
	err := adapter.unmarshaller.Unmarshal(ctx, reqObj, httpReq)
	if err != nil {
		err = adapter.marshaller.Marshal(ctx, httpRespWriter, plzio.Response{nil, err})
		if err != nil {
			ctx.Error("event!failed to write response out", "err", err)
		}
		return
	}
	resp, err := adapter.handler(ctx, ptrReq)
	respObj := adapter.handlerTypeInfo.ResponseBoxer(resp)
	err = adapter.marshaller.Marshal(ctx, httpRespWriter, plzio.Response{respObj, err})
	if err != nil {
		ctx.Error("event!failed to write response out", "err", err)
	}
}

type httpServerUnmarshaller struct {
	reqUnmarshaller plzio.Unmarshaller
}

func (unmarshaller *httpServerUnmarshaller) Unmarshal(ctx *countlog.Context, request interface{}, input interface{}) error {
	httpReq := input.(*http.Request)
	reqBody, err := ioutil.ReadAll(httpReq.Body)
	ctx.TraceCall("callee!ioutil.ReadAll", err)
	if err != nil {
		return err
	}
	ctx.Debug("event!http request", "request", reqBody)
	return unmarshaller.reqUnmarshaller.Unmarshal(ctx, request, reqBody)
}

type httpServerMarshaller struct {
	respMarshaller plzio.Marshaller
}

func (marshaller *httpServerMarshaller) Marshal(ctx *countlog.Context, output interface{}, obj interface{}) error {
	var buf []byte
	err := marshaller.respMarshaller.Marshal(ctx, &buf, obj)
	if err != nil {
		return err
	}
	ctx.Debug("event!http response", "response", buf)
	httpRespWriter := output.(http.ResponseWriter)
	_, err = httpRespWriter.Write(buf)
	return err
}
