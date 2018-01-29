package http

import (
	"github.com/json-iterator/go"
	"github.com/v2pro/plz/countlog"
	"github.com/v2pro/plz/service"
)

type jsoniterUnmarshaller struct {
}

func newJsoniterUnmarshaller() service.Unmarshaller {
	return &jsoniterUnmarshaller{}
}

func (unmarshaller *jsoniterUnmarshaller) Unmarshal(ctx countlog.Context, obj interface{}, input interface{}) error {
	return jsoniter.Unmarshal(input.([]byte), obj)
}
