package http

import (
	"github.com/v2pro/plz/plzio"
	"github.com/v2pro/plz/countlog"
	"unsafe"
	"net/http"
	"bytes"
	"io/ioutil"
)

type Client struct {
	*http.Client
	Unmarshaller plzio.Unmarshaller
	Marshaller   plzio.Marshaller
}

func NewClient(client *http.Client) *Client {
	return &Client{
		Client:       client,
		Unmarshaller: &httpClientUnmarshaller{plzio.NewJsoniterUnmarshaller()},
		Marshaller:   &httpClientMarshaller{plzio.NewJsoniterMarshaller()},
	}
}

func (client *Client) Handle(method string, url string, ptrHandlerObj interface{}) {
	ptrHandler, handlerTypeInfo := plzio.ConvertPtrHandler(ptrHandlerObj)
	*ptrHandler = func(ctx *countlog.Context, ptrReq unsafe.Pointer) (unsafe.Pointer, error) {
		reqObj := handlerTypeInfo.RequestBoxer(ptrReq)
		httpReq, err := http.NewRequest(method, url, nil)
		if err != nil {
			return nil, err
		}
		err = client.Marshaller.Marshal(ctx, httpReq, reqObj)
		if err != nil {
			return nil, err
		}
		httpResp, err := client.Do(httpReq)
		if err != nil {
			return nil, err
		}
		var resp unsafe.Pointer
		ptrResp := unsafe.Pointer(&resp)
		respObj := handlerTypeInfo.ResponseBoxer(ptrResp)
		err = client.Unmarshaller.Unmarshal(ctx, respObj, httpResp)
		if err != nil {
			return nil, err
		}
		return ptrResp, nil
	}
}

type httpClientMarshaller struct {
	reqMarshaller plzio.Marshaller
}

func (marshaller *httpClientMarshaller) Marshal(ctx *countlog.Context, output interface{}, obj interface{}) error {
	var buf []byte
	err := marshaller.reqMarshaller.Marshal(ctx, &buf, obj)
	if err != nil {
		return err
	}
	httpReq := output.(*http.Request)
	httpReq.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
	return nil
}

type httpClientUnmarshaller struct {
	respUnmarshaller plzio.Unmarshaller
}

func (unmarshaller *httpClientUnmarshaller) Unmarshal(ctx *countlog.Context, obj interface{}, input interface{}) error {
	respBody, err := ioutil.ReadAll(input.(*http.Response).Body)
	if err != nil {
		return err
	}
	err = unmarshaller.respUnmarshaller.Unmarshal(ctx, obj, respBody)
	if err != nil {
		return err
	}
	return nil
}
