package core

import (
	"github.com/xukgo/gfs/iService"
	"net/http"
)

func Start(confRepo iService.IConfigRepo) error {
	if confRepo.GetSupportGroupManage() {
		staticHandler = http.StripPrefix("/"+confRepo.GetGroup()+"/", http.FileServer(http.Dir(confRepo.GetStoreDir())))
	} else {
		staticHandler = http.StripPrefix("/", http.FileServer(http.Dir(confRepo.GetStoreDir())))
	}
	Singleton = NewServer(confRepo)
	Singleton.initComponent(false)
	Singleton.Start()
	return nil
}
