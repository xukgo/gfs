package configRepo

import (
	"fmt"
	log "github.com/sjqzhang/seelog"
	"github.com/xukgo/gfs/constDefine"
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
	xmlRoot := new(XmlRoot)
	err = xmlRoot.FillWithXml(contents)
	if err != nil {
		return fmt.Errorf("configRepo unmarshal xml error:%w", err)
	}

	repo := initParamRepo(xmlRoot)
	locker.Lock()
	singleton = repo
	locker.Unlock()
	return nil
}

func initParamRepo(root *XmlRoot) *Repo {
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
	repo := new(Repo)
	repo.Addr = fmt.Sprintf(":%d", root.Port)
	repo.Host = fmt.Sprintf("http://%s:%d", root.IP, root.Port)
	repo.PeerId = fmt.Sprintf("%d", root.NodeId)
	repo.Peers = root.Peers

	repo.ShowDir = root.ShowDir
	repo.EnableHttps = false
	repo.SupportGroupManage = false
	repo.EnableWebUpload = true
	repo.RefreshInterval = 1800
	repo.EnableCustomPath = true
	repo.AutoRepair = true
	repo.FileSumArithmetic = "md5"
	repo.AdminIps = []string{"127.0.0.1"}
	repo.EnableDistinctFile = true
	repo.EnableCrossOrigin = true
	repo.EnableDownloadAuth = true
	repo.EnableTus = true
	repo.SyncTimeout = 0

	repo.QueueSize = constDefine.CONST_QUEUE_SIZE
	repo.ReadTimeout = 60 * 10
	repo.WriteTimeout = 60 * 10
	repo.SyncWorker = 200
	repo.UploadWorker = runtime.NumCPU() + 4
	if runtime.NumCPU() < 4 {
		repo.UploadWorker = 8
	}
	repo.UploadQueueSize = 200
	repo.RetryCount = 3
	repo.SyncDelay = 60
	repo.WatchChanSize = 100000

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
	return repo
}
