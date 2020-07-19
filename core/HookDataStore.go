package core

import (
	"errors"
	"fmt"
	"github.com/astaxie/beego/httplib"
	log "github.com/sjqzhang/seelog"
	"github.com/sjqzhang/tusd"
	"github.com/xukgo/gfs/model"
	"strings"
	"time"
)

type HookDataStore struct {
	tusd.DataStore
}

func (store HookDataStore) NewUpload(info tusd.FileInfo) (id string, err error) {
	var (
		jsonResult model.JsonResult
	)
	if Config().AuthUrl != "" {
		if auth_token, ok := info.MetaData["auth_token"]; !ok {
			msg := "token auth fail,auth_token is not in http header Upload-Metadata," +
				"in uppy uppy.setMeta({ auth_token: '9ee60e59-cb0f-4578-aaba-29b9fc2919ca' })"
			log.Error(msg, fmt.Sprintf("current header:%v", info.MetaData))
			return "", model.InitHttpError(errors.New(msg), 401)
		} else {
			req := httplib.Post(Config().AuthUrl)
			req.Param("auth_token", auth_token)
			req.SetTimeout(time.Second*5, time.Second*10)
			content, err := req.String()
			content = strings.TrimSpace(content)
			if strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}") {
				if err = json.Unmarshal([]byte(content), &jsonResult); err != nil {
					log.Error(err)
					return "", model.InitHttpError(errors.New(err.Error()+content), 401)
				}
				if jsonResult.Data != "ok" {
					return "", model.InitHttpError(errors.New(content), 401)
				}
			} else {
				if err != nil {
					log.Error(err)
					return "", err
				}
				if strings.TrimSpace(content) != "ok" {
					return "", model.InitHttpError(errors.New(content), 401)
				}
			}
		}
	}
	return store.DataStore.NewUpload(info)
}
