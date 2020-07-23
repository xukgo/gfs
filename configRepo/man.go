package configRepo

import (
	"fmt"
	"github.com/xukgo/gfs/constDefine"
	"github.com/xukgo/gsaber/utils/randomUtil"
	"io/ioutil"
	"runtime"
	"sync"
)

var locker sync.RWMutex
var singleton *Repo = nil

func GetSingleton() *Repo {
	locker.RLock()
	v := singleton
	locker.RUnlock()
	return v
}

func InitRepo(filePath string) error {
	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("ReadFile %s error", filePath)
	}
	repo := new(Repo)
	err = repo.FillWithJson(contents)
	if err != nil {
		return fmt.Errorf("configRepo unmarshal json error:%w", err)
	}
	afterFillJson(repo)
	locker.Lock()
	singleton = repo
	locker.Unlock()
	return nil
}

func afterFillJson(repo *Repo) {
	/*
		if !Singleton.util.FileExists(CONF_FILE_NAME) {
				var ip string
				if ip = os.Getenv("GFS_IP"); ip == "" {
					ip = Singleton.util.GetPulicIP()
				}
				peer := "http://" + ip + ":8080"
				cfg := fmt.Sprintf(constDefine.CONF_JSON_TEMPLATE, peerId, peer, peer)
				Singleton.util.WriteFile(CONF_FILE_NAME, cfg)
			}
	*/
	if repo.QueueSize == 0 {
		repo.QueueSize = constDefine.CONST_QUEUE_SIZE
	}
	if repo.PeerId == "" {
		repo.PeerId = fmt.Sprintf("%d", randomUtil.NewInt32(0, 9))
	}
	if repo.ReadTimeout == 0 {
		repo.ReadTimeout = 60 * 10
	}
	if repo.WriteTimeout == 0 {
		repo.WriteTimeout = 60 * 10
	}
	if repo.SyncWorker == 0 {
		repo.SyncWorker = 200
	}
	if repo.UploadWorker == 0 {
		repo.UploadWorker = runtime.NumCPU() + 4
		if runtime.NumCPU() < 4 {
			repo.UploadWorker = 8
		}
	}
	if repo.UploadQueueSize == 0 {
		repo.UploadQueueSize = 200
	}
	if repo.RetryCount == 0 {
		repo.RetryCount = 3
	}
	if repo.SyncDelay == 0 {
		repo.SyncDelay = 60
	}
	if repo.WatchChanSize == 0 {
		repo.WatchChanSize = 100000
	}

	if repo.Host == "" {
		//todo 自动生成局域网内到http url
		repo.Host = "http://ip:" + repo.Addr
	}
}
