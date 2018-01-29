package http

import (
	"github.com/json-iterator/go"
	"github.com/v2pro/plz/countlog"
	"github.com/v2pro/plz/service"
)

type jsoniterUnmarshaller struct {
}


func (unmarshaller *jsoniterUnmarshaller) Unmarshal(ctx *countlog.Context, obj interface{}, input interface{}) error {
	return jsoniter.Unmarshal(input.([]byte), obj)
}

type jsoniterResponseUnmarshaller struct {
}

func (unmarshaller *jsoniterResponseUnmarshaller) Unmarshal(ctx *countlog.Context, obj interface{}, input interface{}) error {
	resp := obj.(*service.Response)
	iter := jsoniter.ConfigDefault.BorrowIterator(input.([]byte))
	defer jsoniter.ConfigDefault.ReturnIterator(iter)
	err := &service.WithNumberError{}
	iter.ReadObjectCB(func(iterator *jsoniter.Iterator, field string) bool {
		switch field {
		case "errno":
			err.Number = iterator.ReadInt()
		case "errmsg":
			err.Message = iterator.ReadString()
		case "data":
			iterator.ReadVal(resp.Object)
		default:
			iterator.Skip()
		}
		return true
	})
	if err.Message != "" {
		resp.Error = err
	}
	return iter.Error
}
