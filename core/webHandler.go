package core

import (
	log "github.com/sjqzhang/seelog"
	"io"
	"net/http"
	"os"
	"time"
)

func (this *Server) Upload(w http.ResponseWriter, r *http.Request) {
	var (
		err    error
		fn     string
		folder string
		fpTmp  *os.File
		fpBody *os.File
	)
	if r.Method == http.MethodGet {
		this.upload(w, r)
		return
	}
	folder = STORE_DIR + "/_tmp/" + time.Now().Format("20060102")
	if !this.util.FileExists(folder) {
		if err = os.MkdirAll(folder, 0777); err != nil {
			log.Error(err)
		}
	}
	fn = folder + "/" + this.util.GetUUID()
	defer func() {
		os.Remove(fn)
	}()
	fpTmp, err = os.OpenFile(fn, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		log.Error(err)
		w.Write([]byte(err.Error()))
		return
	}
	defer fpTmp.Close()
	if _, err = io.Copy(fpTmp, r.Body); err != nil {
		log.Error(err)
		w.Write([]byte(err.Error()))
		return
	}
	fpBody, err = os.Open(fn)
	r.Body = fpBody
	done := make(chan bool, 1)
	this.queueUpload <- WrapReqResp{&w, r, done}
	<-done
}
