package model

import "net/http"

type WrapReqResp struct {
	W    *http.ResponseWriter
	R    *http.Request
	Done chan bool
}
