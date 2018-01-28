package http

import (
	"net/http"
	"github.com/v2pro/plz/countlog"
	"io/ioutil"
	"unsafe"
	"github.com/v2pro/plz/plzio"
)

type Server struct {
	mux *http.ServeMux
}

func NewServer() *Server {
	mux := &http.ServeMux{}
	return &Server{
		mux: mux,
	}
}

func (server *Server) Start(addr string) {
	http.ListenAndServe(addr, server.mux)
}

func (server *Server) Handle(pattern string, handlerObj interface{}) {
	handler, handlerTypeInfo := plzio.ConvertHandler(handlerObj)
	server.mux.Handle(pattern, &handlerAdapter{
		unmarshaller:    &httpUnmarshaller{plzio.NewJsoniterUnmarshaller()},
		marshaller:      &httpMarshaller{plzio.NewJsoniterMarshaller()},
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
		err = adapter.marshaller.Marshal(ctx, httpRespWriter, nil, err)
		if err != nil {
			ctx.Error("event!failed to write response out", "err", err)
		}
		return
	}
	resp, err := adapter.handler(ctx, ptrReq)
	respObj := adapter.handlerTypeInfo.ResponseBoxer(resp)
	err = adapter.marshaller.Marshal(ctx, httpRespWriter, respObj, err)
	if err != nil {
		ctx.Error("event!failed to write response out", "err", err)
	}
}

type httpUnmarshaller struct {
	reqUnmarshaller plzio.Unmarshaller
}

func (unmarshaller *httpUnmarshaller) Unmarshal(ctx *countlog.Context, request interface{}, input interface{}) error {
	httpReq := input.(*http.Request)
	reqBody, err := ioutil.ReadAll(httpReq.Body)
	ctx.TraceCall("callee!ioutil.ReadAll", err)
	if err != nil {
		return err
	}
	ctx.Debug("event!http request", "request", reqBody)
	return unmarshaller.reqUnmarshaller.Unmarshal(ctx, request, reqBody)
}

type httpMarshaller struct {
	respMarshaller plzio.Marshaller
}

func (marshaller *httpMarshaller) Marshal(ctx *countlog.Context, output interface{}, response interface{}, responseError error) error {
	var buf []byte
	err := marshaller.respMarshaller.Marshal(ctx, &buf, response, responseError)
	if err != nil {
		return err
	}
	ctx.Debug("event!http response", "response", buf)
	httpRespWriter := output.(http.ResponseWriter)
	_, err = httpRespWriter.Write(buf)
	return err
}
