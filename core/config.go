package core

import (
	"fmt"
	log "github.com/sjqzhang/seelog"
	"github.com/xukgo/gfs/configRepo"
	"github.com/xukgo/gfs/constDefine"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"unsafe"
)

func init() {
	appDir, e1 := filepath.Abs(filepath.Dir(os.Args[0]))
	curDir, e2 := filepath.Abs(".")
	if e1 == nil && e2 == nil && appDir != curDir && !strings.Contains(appDir, "go-build") {
		msg := fmt.Sprintf("please change directory to '%s' start fileserver\n", appDir)
		msg = msg + fmt.Sprintf("请切换到 '%s' 目录启动 fileserver ", appDir)
		log.Warn(msg)
		fmt.Println(msg)
		os.Exit(1)
	}
	DOCKER_DIR = os.Getenv("GFS_DIR")
	if DOCKER_DIR != "" {
		if !strings.HasSuffix(DOCKER_DIR, "/") {
			DOCKER_DIR = DOCKER_DIR + "/"
		}
	}
	STORE_DIR = DOCKER_DIR + constDefine.STORE_DIR_NAME
	CONF_DIR = DOCKER_DIR + constDefine.CONF_DIR_NAME
	DATA_DIR = DOCKER_DIR + constDefine.DATA_DIR_NAME
	LOG_DIR = DOCKER_DIR + constDefine.LOG_DIR_NAME
	STATIC_DIR = DOCKER_DIR + constDefine.STATIC_DIR_NAME
	LARGE_DIR_NAME = "haystack"
	LARGE_DIR = STORE_DIR + "/haystack"
	LEVELDB_FILE_NAME = DATA_DIR + "/fileserver.db"
	LOG_LEVELDB_FILE_NAME = DATA_DIR + "/log.db"
	STAT_FILE_NAME = DATA_DIR + "/stat.json"
	CONF_FILE_NAME = CONF_DIR + "/cfg.json"
	SERVER_CRT_FILE_NAME = CONF_DIR + "/server.crt"
	SERVER_KEY_FILE_NAME = CONF_DIR + "/server.key"
	SEARCH_FILE_NAME = DATA_DIR + "/search.txt"
	FOLDERS = []string{DATA_DIR, STORE_DIR, CONF_DIR, STATIC_DIR}
	logAccessConfigStr = strings.Replace(constDefine.LOG_ACCESS_CONF_TEMPLATE, "{DOCKER_DIR}", DOCKER_DIR, -1)
	logConfigStr = strings.Replace(constDefine.LOG_CONF_TEMPLATE, "{DOCKER_DIR}", DOCKER_DIR, -1)
	for _, folder := range FOLDERS {
		os.MkdirAll(folder, 0775)
	}
	Singleton = NewServer()

	peerId := fmt.Sprintf("%d", Singleton.util.RandInt(0, 9))
	if !Singleton.util.FileExists(CONF_FILE_NAME) {
		var ip string
		if ip = os.Getenv("GFS_IP"); ip == "" {
			ip = Singleton.util.GetPulicIP()
		}
		peer := "http://" + ip + ":8080"
		cfg := fmt.Sprintf(constDefine.CONF_JSON_TEMPLATE, peerId, peer, peer)
		Singleton.util.WriteFile(CONF_FILE_NAME, cfg)
	}
	if logger, err := log.LoggerFromConfigAsBytes([]byte(logConfigStr)); err != nil {
		panic(err)
	} else {
		log.ReplaceLogger(logger)
	}
	if _logacc, err := log.LoggerFromConfigAsBytes([]byte(logAccessConfigStr)); err == nil {
		logacc = _logacc
		log.Info("succes init log access")
	} else {
		log.Error(err.Error())
	}
	ParseConfig(CONF_FILE_NAME)
	if Config().QueueSize == 0 {
		Config().QueueSize = constDefine.CONST_QUEUE_SIZE
	}
	if Config().PeerId == "" {
		Config().PeerId = peerId
	}
	if Config().SupportGroupManage {
		staticHandler = http.StripPrefix("/"+Config().Group+"/", http.FileServer(http.Dir(STORE_DIR)))
	} else {
		staticHandler = http.StripPrefix("/", http.FileServer(http.Dir(STORE_DIR)))
	}
	Singleton.initComponent(false)
}

func Config() *configRepo.GloablConfig {
	cnf := (*configRepo.GloablConfig)(atomic.LoadPointer(&ptr))
	return cnf
}
func ParseConfig(filePath string) {
	var (
		data []byte
	)
	if filePath == "" {
		data = []byte(strings.TrimSpace(constDefine.CONF_JSON_TEMPLATE))
	} else {
		file, err := os.Open(filePath)
		if err != nil {
			panic(fmt.Sprintln("open file path:", filePath, "error:", err))
		}
		defer file.Close()
		FileName = filePath
		data, err = ioutil.ReadAll(file)
		if err != nil {
			panic(fmt.Sprintln("file path:", filePath, " read all error:", err))
		}
	}
	var c configRepo.GloablConfig
	if err := json.Unmarshal(data, &c); err != nil {
		panic(fmt.Sprintln("file path:", filePath, "json unmarshal error:", err))
	}
	log.Info(c)
	atomic.StorePointer(&ptr, unsafe.Pointer(&c))
	log.Info("config parse success")
}
