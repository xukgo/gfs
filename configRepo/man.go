package configRepo

import (
	"fmt"
	log "github.com/sjqzhang/seelog"
	"github.com/xukgo/gfs/constDefine"
	"github.com/xukgo/gsaber/utils/randomUtil"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
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

	repo.DockerDir = os.Getenv("GFS_DIR")
	if repo.DockerDir != "" {
		if !strings.HasSuffix(repo.DockerDir, "/") {
			repo.DockerDir = repo.DockerDir + "/"
		}
	}
	repo.StoreDir = repo.DockerDir + constDefine.STORE_DIR_NAME
	repo.ConfDir = repo.DockerDir + constDefine.CONF_DIR_NAME
	repo.DataDir = repo.DockerDir + constDefine.DATA_DIR_NAME
	repo.LogDir = repo.DockerDir + constDefine.LOG_DIR_NAME
	repo.StaticDir = repo.DockerDir + constDefine.STATIC_DIR_NAME
	repo.LargeDirName = "haystack"
	repo.LargeDir = repo.StoreDir + "/" + repo.LargeDirName
	repo.LevelDbFileName = repo.DataDir + "/fileserver.db"
	repo.LogLevelDbFileName = repo.DataDir + "/log.db"
	repo.StatFileName = repo.DataDir + "/stat.json"
	repo.ConfFileName = repo.ConfDir + "/cfg.json"
	repo.ServerCrtFileName = repo.ConfDir + "/server.crt"
	repo.ServerKeyFileName = repo.ConfDir + "/server.key"
	repo.SearchFileName = repo.DataDir + "/search.txt"

	folders := []string{repo.DataDir, repo.StoreDir, repo.ConfDir, repo.StaticDir}
	for _, folder := range folders {
		os.MkdirAll(folder, 0775)
	}
	logAccessConfigStr := strings.Replace(constDefine.LOG_ACCESS_CONF_TEMPLATE, "{DOCKER_DIR}", repo.DockerDir, -1)
	logConfigStr := strings.Replace(constDefine.LOG_CONF_TEMPLATE, "{DOCKER_DIR}", repo.DockerDir, -1)
	if logger, err := log.LoggerFromConfigAsBytes([]byte(logConfigStr)); err != nil {
		panic(err)
	} else {
		log.ReplaceLogger(logger)
	}
	if _logacc, err := log.LoggerFromConfigAsBytes([]byte(logAccessConfigStr)); err == nil {
		//logacc := _logacc
		_ = _logacc
		log.Info("succes init log access")
	} else {
		log.Error(err.Error())
	}
}
