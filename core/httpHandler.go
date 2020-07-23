package core

import (
	"fmt"
	log "github.com/sjqzhang/seelog"
	"net/http"
	"runtime/debug"
	"time"
)

type HttpHandler struct {
	EnableCrossOrigin bool
}

func (this *HttpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	status_code := "200"
	defer func(t time.Time) {
		logStr := fmt.Sprintf("[Access] %s | %s | %s | %s | %s |%s",
			time.Now().Format("2006/01/02 - 15:04:05"),
			//res.Header(),
			time.Since(t).String(),
			Singleton.util.GetClientIp(req),
			req.Method,
			status_code,
			req.RequestURI,
		)
		logacc.Info(logStr)
	}(time.Now())
	defer func() {
		if err := recover(); err != nil {
			status_code = "500"
			res.WriteHeader(500)
			print(err)
			buff := debug.Stack()
			log.Error(err)
			log.Error(string(buff))
		}
	}()
	if this.EnableCrossOrigin {
		Singleton.CrossOrigin(res, req)
	}
	http.DefaultServeMux.ServeHTTP(res, req)
}
