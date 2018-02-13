package http

import (
	"github.com/v2pro/plz/countlog"
	"io/ioutil"
	"net/http"
	"unsafe"
	"github.com/v2pro/plz.service/service"
)

type Server struct {
	Unmarshaller service.Unmarshaller
	Marshaller   service.Marshaller
	mux          *http.ServeMux
	server       *http.Server
}

func NewServer() *Server {
	mux := &http.ServeMux{}
	return &Server{
		Unmarshaller: &httpServerUnmarshaller{&jsoniterUnmarshaller{}},
		Marshaller:   &httpServerMarshaller{&jsoniterResponseMarshaller{}},
		mux:          mux,
	}
}

func (server *Server) Start(addr string) error {
	srv := &http.Server{Addr: addr, Handler: server.mux}
	server.server = srv
	return srv.ListenAndServe()
}

func (server *Server) Shutdown(ctx *countlog.Context) error {
	if server.server == nil {
		return nil
	}
	return server.server.Shutdown(ctx)
}

func (server *Server) Close() error {
	if server.server == nil {
		return nil
	}
	return server.server.Close()
}

func (server *Server) Handle(pattern string, handlerObj interface{}) {
	handler, handlerTypeInfo := service.ConvertHandler(handlerObj)
	server.mux.Handle(pattern, &handlerAdapter{
		marshaller:      server.Marshaller,
		unmarshaller:    server.Unmarshaller,
		handler:         handler,
		handlerTypeInfo: handlerTypeInfo,
	})
}

type handlerAdapter struct {
	unmarshaller    service.Unmarshaller
	marshaller      service.Marshaller
	handler         service.Handler
	handlerTypeInfo *service.HandlerTypeInfo
}

func (adapter *handlerAdapter) ServeHTTP(httpRespWriter http.ResponseWriter, httpReq *http.Request) {
	ctxObj := httpReq.Context()
	ctx := countlog.Ctx(ctxObj)
	var req unsafe.Pointer
	ptrReq := unsafe.Pointer(&req)
	reqObj := adapter.handlerTypeInfo.RequestBoxer(ptrReq)
	err := adapter.unmarshaller.Unmarshal(ctx, reqObj, httpReq)
	if err != nil {
		err = adapter.marshaller.Marshal(ctx, httpRespWriter, service.Response{nil, err})
		if err != nil {
			ctx.Error("event!failed to write response out", "err", err)
		}
		return
	}
	resp, err := adapter.handler(ctx, ptrReq)
	respObj := adapter.handlerTypeInfo.ResponseBoxer(resp)
	err = adapter.marshaller.Marshal(ctx, httpRespWriter, service.Response{respObj, err})
	if err != nil {
		ctx.Error("event!failed to write response out", "err", err)
	}
}

type httpServerUnmarshaller struct {
	reqUnmarshaller service.Unmarshaller
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
	respMarshaller service.Marshaller
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
