package core

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/xukgo/gfs/configRepo"
	"github.com/xukgo/gfs/constDefine"
	"github.com/xukgo/gfs/model"
	"io"
	"io/ioutil"
	slog "log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/astaxie/beego/httplib"
	mapset "github.com/deckarep/golang-set"
	_ "github.com/eventials/go-tus"
	jsoniter "github.com/json-iterator/go"
	"github.com/sjqzhang/goutil"
	log "github.com/sjqzhang/seelog"
	"github.com/sjqzhang/tusd"
	"github.com/sjqzhang/tusd/filestore"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var staticHandler http.Handler
var json = jsoniter.ConfigCompatibleWithStandardLibrary
var Singleton *Server = nil
var logacc log.LoggerInterface
var FOLDERS = []string{DATA_DIR, STORE_DIR, CONF_DIR, STATIC_DIR}

var (
	logConfigStr             string
	logAccessConfigStr       string
	FileName                 string
	ptr                      unsafe.Pointer
	DOCKER_DIR               = ""
	STORE_DIR                = constDefine.STORE_DIR_NAME
	CONF_DIR                 = constDefine.CONF_DIR_NAME
	LOG_DIR                  = constDefine.LOG_DIR_NAME
	DATA_DIR                 = constDefine.DATA_DIR_NAME
	STATIC_DIR               = constDefine.STATIC_DIR_NAME
	LARGE_DIR_NAME           = "haystack"
	LARGE_DIR                = STORE_DIR + "/haystack"
	LEVELDB_FILE_NAME        = DATA_DIR + "/fileserver.db"
	LOG_LEVELDB_FILE_NAME    = DATA_DIR + "/log.db"
	STAT_FILE_NAME           = DATA_DIR + "/stat.json"
	CONF_FILE_NAME           = CONF_DIR + "/cfg.json"
	SERVER_CRT_FILE_NAME     = CONF_DIR + "/server.crt"
	SERVER_KEY_FILE_NAME     = CONF_DIR + "/server.key"
	SEARCH_FILE_NAME         = DATA_DIR + "/search.txt"
	CONST_UPLOAD_COUNTER_KEY = "__CONST_UPLOAD_COUNTER_KEY__"
)

type Server struct {
	ldb            *leveldb.DB
	logDB          *leveldb.DB
	util           *goutil.Common
	statMap        *goutil.CommonMap
	sumMap         *goutil.CommonMap
	rtMap          *goutil.CommonMap
	queueToPeers   chan model.FileInfo
	queueFromPeers chan model.FileInfo
	queueFileLog   chan *model.FileLog
	queueUpload    chan model.WrapReqResp
	lockMap        *goutil.CommonMap
	sceneMap       *goutil.CommonMap
	searchMap      *goutil.CommonMap
	curDate        string
	host           string
}

func NewServer() *Server {
	var (
		//Singleton *Server
		err error
	)
	if Singleton != nil {
		return Singleton
	}
	Singleton = &Server{
		util:           &goutil.Common{},
		statMap:        goutil.NewCommonMap(0),
		lockMap:        goutil.NewCommonMap(0),
		rtMap:          goutil.NewCommonMap(0),
		sceneMap:       goutil.NewCommonMap(0),
		searchMap:      goutil.NewCommonMap(0),
		queueToPeers:   make(chan model.FileInfo, constDefine.CONST_QUEUE_SIZE),
		queueFromPeers: make(chan model.FileInfo, constDefine.CONST_QUEUE_SIZE),
		queueFileLog:   make(chan *model.FileLog, constDefine.CONST_QUEUE_SIZE),
		queueUpload:    make(chan model.WrapReqResp, 100),
		sumMap:         goutil.NewCommonMap(365 * 3),
	}

	defaultTransport := &http.Transport{
		DisableKeepAlives:   true,
		Dial:                httplib.TimeoutDialer(time.Second*15, time.Second*300),
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
	}
	settins := httplib.BeegoHTTPSettings{
		UserAgent:        "GFS",
		ConnectTimeout:   15 * time.Second,
		ReadWriteTimeout: 15 * time.Second,
		Gzip:             true,
		DumpBody:         true,
		Transport:        defaultTransport,
	}
	httplib.SetDefaultSetting(settins)
	Singleton.statMap.Put(constDefine.CONST_STAT_FILE_COUNT_KEY, int64(0))
	Singleton.statMap.Put(constDefine.CONST_STAT_FILE_TOTAL_SIZE_KEY, int64(0))
	Singleton.statMap.Put(Singleton.util.GetToDay()+"_"+constDefine.CONST_STAT_FILE_COUNT_KEY, int64(0))
	Singleton.statMap.Put(Singleton.util.GetToDay()+"_"+constDefine.CONST_STAT_FILE_TOTAL_SIZE_KEY, int64(0))
	Singleton.curDate = Singleton.util.GetToDay()
	opts := &opt.Options{
		CompactionTableSize: 1024 * 1024 * 20,
		WriteBuffer:         1024 * 1024 * 20,
	}
	Singleton.ldb, err = leveldb.OpenFile(LEVELDB_FILE_NAME, opts)
	if err != nil {
		fmt.Println(fmt.Sprintf("open db file %s fail,maybe has opening", LEVELDB_FILE_NAME))
		log.Error(err)
		panic(err)
	}
	Singleton.logDB, err = leveldb.OpenFile(LOG_LEVELDB_FILE_NAME, opts)
	if err != nil {
		fmt.Println(fmt.Sprintf("open db file %s fail,maybe has opening", LOG_LEVELDB_FILE_NAME))
		log.Error(err)
		panic(err)

	}
	return Singleton
}
func (this *Server) GetServerURI(r *http.Request) string {
	return fmt.Sprintf("http://%s/", r.Host)
}

func (this *Server) GetFilePathByInfo(fileInfo *model.FileInfo, withDocker bool) string {
	var (
		fn string
	)
	fn = fileInfo.Name
	if fileInfo.ReName != "" {
		fn = fileInfo.ReName
	}
	if withDocker {
		return DOCKER_DIR + fileInfo.Path + "/" + fn
	}
	return fileInfo.Path + "/" + fn
}

func (this *Server) CheckFileAndSendToPeer(date string, filename string, isForceUpload bool) {
	var (
		md5set mapset.Set
		err    error
		md5s   []interface{}
	)
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			log.Error("CheckFileAndSendToPeer")
			log.Error(re)
			log.Error(string(buffer))
		}
	}()
	if md5set, err = this.GetMd5sByDate(date, filename); err != nil {
		log.Error(err)
		return
	}
	md5s = md5set.ToSlice()
	for _, md := range md5s {
		if md == nil {
			continue
		}
		if fileInfo, _ := this.GetFileInfoFromLevelDB(md.(string)); fileInfo != nil && fileInfo.Md5 != "" {
			if isForceUpload {
				fileInfo.Peers = []string{}
			}
			if len(fileInfo.Peers) > len(Config().Peers) {
				continue
			}
			if !this.util.Contains(this.host, fileInfo.Peers) {
				fileInfo.Peers = append(fileInfo.Peers, this.host) // peer is null
			}
			if filename == constDefine.CONST_Md5_QUEUE_FILE_NAME {
				this.AppendToDownloadQueue(fileInfo)
			} else {
				this.AppendToQueue(fileInfo)
			}
		}
	}
}
func (this *Server) postFileToPeer(fileInfo *model.FileInfo) {
	var (
		err      error
		peer     string
		filename string
		info     *model.FileInfo
		postURL  string
		result   string
		fi       os.FileInfo
		i        int
		data     []byte
		fpath    string
	)
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			log.Error("postFileToPeer")
			log.Error(re)
			log.Error(string(buffer))
		}
	}()
	//fmt.Println("postFile",fileInfo)
	for i, peer = range Config().Peers {
		_ = i
		if fileInfo.Peers == nil {
			fileInfo.Peers = []string{}
		}
		if this.util.Contains(peer, fileInfo.Peers) {
			continue
		}
		filename = fileInfo.Name
		if fileInfo.ReName != "" {
			filename = fileInfo.ReName
			if fileInfo.OffSet != -1 {
				filename = strings.Split(fileInfo.ReName, ",")[0]
			}
		}
		fpath = DOCKER_DIR + fileInfo.Path + "/" + filename
		if !this.util.FileExists(fpath) {
			log.Warn(fmt.Sprintf("file '%s' not found", fpath))
			continue
		} else {
			if fileInfo.Size == 0 {
				if fi, err = os.Stat(fpath); err != nil {
					log.Error(err)
				} else {
					fileInfo.Size = fi.Size()
				}
			}
		}
		if fileInfo.OffSet != -2 && Config().EnableDistinctFile {
			//not migrate file should check or update file
			// where not EnableDistinctFile should check
			if info, err = this.checkPeerFileExist(peer, fileInfo.Md5, ""); info.Md5 != "" {
				fileInfo.Peers = append(fileInfo.Peers, peer)
				if _, err = this.SaveFileInfoToLevelDB(fileInfo.Md5, fileInfo, this.ldb); err != nil {
					log.Error(err)
				}
				continue
			}
		}
		postURL = fmt.Sprintf("%s%s", peer, this.getRequestURI("syncfile_info"))
		b := httplib.Post(postURL)
		b.SetTimeout(time.Second*30, time.Second*30)
		if data, err = json.Marshal(fileInfo); err != nil {
			log.Error(err)
			return
		}
		b.Param("fileInfo", string(data))
		result, err = b.String()
		if err != nil {
			if fileInfo.Retry <= Config().RetryCount {
				fileInfo.Retry = fileInfo.Retry + 1
				this.AppendToQueue(fileInfo)
			}
			log.Error(err, fmt.Sprintf(" path:%s", fileInfo.Path+"/"+fileInfo.Name))
		}
		if !strings.HasPrefix(result, "http://") || err != nil {
			this.SaveFileMd5Log(fileInfo, constDefine.CONST_Md5_ERROR_FILE_NAME)
		}
		if strings.HasPrefix(result, "http://") {
			log.Info(result)
			if !this.util.Contains(peer, fileInfo.Peers) {
				fileInfo.Peers = append(fileInfo.Peers, peer)
				if _, err = this.SaveFileInfoToLevelDB(fileInfo.Md5, fileInfo, this.ldb); err != nil {
					log.Error(err)
				}
			}
		}
		if err != nil {
			log.Error(err)
		}
	}
}
func (this *Server) SaveFileMd5Log(fileInfo *model.FileInfo, filename string) {
	var (
		info model.FileInfo
	)
	for len(this.queueFileLog)+len(this.queueFileLog)/10 > constDefine.CONST_QUEUE_SIZE {
		time.Sleep(time.Second * 1)
	}
	info = *fileInfo
	this.queueFileLog <- &model.FileLog{FileInfo: &info, FileName: filename}
}
func (this *Server) saveFileMd5Log(fileInfo *model.FileInfo, filename string) {
	var (
		err      error
		outname  string
		logDate  string
		ok       bool
		fullpath string
		md5Path  string
		logKey   string
	)
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			log.Error("saveFileMd5Log")
			log.Error(re)
			log.Error(string(buffer))
		}
	}()
	if fileInfo == nil || fileInfo.Md5 == "" || filename == "" {
		log.Warn("saveFileMd5Log", fileInfo, filename)
		return
	}
	logDate = this.util.GetDayFromTimeStamp(fileInfo.TimeStamp)
	outname = fileInfo.Name
	if fileInfo.ReName != "" {
		outname = fileInfo.ReName
	}
	fullpath = fileInfo.Path + "/" + outname
	logKey = fmt.Sprintf("%s_%s_%s", logDate, filename, fileInfo.Md5)
	if filename == constDefine.CONST_FILE_Md5_FILE_NAME {
		//this.searchMap.Put(fileInfo.Md5, fileInfo.Name)
		if ok, err = this.IsExistFromLevelDB(fileInfo.Md5, this.ldb); !ok {
			this.statMap.AddCountInt64(logDate+"_"+constDefine.CONST_STAT_FILE_COUNT_KEY, 1)
			this.statMap.AddCountInt64(logDate+"_"+constDefine.CONST_STAT_FILE_TOTAL_SIZE_KEY, fileInfo.Size)
			this.SaveStat()
		}
		if _, err = this.SaveFileInfoToLevelDB(logKey, fileInfo, this.logDB); err != nil {
			log.Error(err)
		}
		if _, err := this.SaveFileInfoToLevelDB(fileInfo.Md5, fileInfo, this.ldb); err != nil {
			log.Error("saveToLevelDB", err, fileInfo)
		}
		if _, err = this.SaveFileInfoToLevelDB(this.util.MD5(fullpath), fileInfo, this.ldb); err != nil {
			log.Error("saveToLevelDB", err, fileInfo)
		}
		return
	}
	if filename == constDefine.CONST_REMOME_Md5_FILE_NAME {
		//this.searchMap.Remove(fileInfo.Md5)
		if ok, err = this.IsExistFromLevelDB(fileInfo.Md5, this.ldb); ok {
			this.statMap.AddCountInt64(logDate+"_"+constDefine.CONST_STAT_FILE_COUNT_KEY, -1)
			this.statMap.AddCountInt64(logDate+"_"+constDefine.CONST_STAT_FILE_TOTAL_SIZE_KEY, -fileInfo.Size)
			this.SaveStat()
		}
		this.RemoveKeyFromLevelDB(logKey, this.logDB)
		md5Path = this.util.MD5(fullpath)
		if err := this.RemoveKeyFromLevelDB(fileInfo.Md5, this.ldb); err != nil {
			log.Error("RemoveKeyFromLevelDB", err, fileInfo)
		}
		if err = this.RemoveKeyFromLevelDB(md5Path, this.ldb); err != nil {
			log.Error("RemoveKeyFromLevelDB", err, fileInfo)
		}
		// remove files.md5 for stat info(repair from logDB)
		logKey = fmt.Sprintf("%s_%s_%s", logDate, constDefine.CONST_FILE_Md5_FILE_NAME, fileInfo.Md5)
		this.RemoveKeyFromLevelDB(logKey, this.logDB)
		return
	}
	this.SaveFileInfoToLevelDB(logKey, fileInfo, this.logDB)
}
func (this *Server) IsExistFromLevelDB(key string, db *leveldb.DB) (bool, error) {
	return db.Has([]byte(key), nil)
}

func (this *Server) checkPeerFileExist(peer string, md5sum string, fpath string) (*model.FileInfo, error) {
	var (
		err      error
		fileInfo model.FileInfo
	)
	req := httplib.Post(fmt.Sprintf("%s%s?md5=%s", peer, this.getRequestURI("check_file_exist"), md5sum))
	req.Param("path", fpath)
	req.Param("md5", md5sum)
	req.SetTimeout(time.Second*5, time.Second*10)
	if err = req.ToJSON(&fileInfo); err != nil {
		return &model.FileInfo{}, err
	}
	if fileInfo.Md5 == "" {
		return &fileInfo, errors.New("not found")
	}
	return &fileInfo, nil
}
func (this *Server) Sync(w http.ResponseWriter, r *http.Request) {
	var (
		result model.JsonResult
	)
	r.ParseForm()
	result.Status = "fail"
	if !this.IsPeer(r) {
		result.Message = "client must be in cluster"
		w.Write([]byte(this.util.JsonEncodePretty(result)))
		return
	}
	date := ""
	force := ""
	inner := ""
	isForceUpload := false
	force = r.FormValue("force")
	date = r.FormValue("date")
	inner = r.FormValue("inner")
	if force == "1" {
		isForceUpload = true
	}
	if inner != "1" {
		for _, peer := range Config().Peers {
			req := httplib.Post(peer + this.getRequestURI("sync"))
			req.Param("force", force)
			req.Param("inner", "1")
			req.Param("date", date)
			if _, err := req.String(); err != nil {
				log.Error(err)
			}
		}
	}
	if date == "" {
		result.Message = "require paramete date &force , ?date=20181230"
		w.Write([]byte(this.util.JsonEncodePretty(result)))
		return
	}
	date = strings.Replace(date, ".", "", -1)
	if isForceUpload {
		go this.CheckFileAndSendToPeer(date, constDefine.CONST_FILE_Md5_FILE_NAME, isForceUpload)
	} else {
		go this.CheckFileAndSendToPeer(date, constDefine.CONST_Md5_ERROR_FILE_NAME, isForceUpload)
	}
	result.Status = "ok"
	result.Message = "job is running"
	w.Write([]byte(this.util.JsonEncodePretty(result)))
}
func (this *Server) GetFileInfoFromLevelDB(key string) (*model.FileInfo, error) {
	var (
		err      error
		data     []byte
		fileInfo model.FileInfo
	)
	if data, err = this.ldb.Get([]byte(key), nil); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &fileInfo); err != nil {
		return nil, err
	}
	return &fileInfo, nil
}
func (this *Server) SaveStat() {
	SaveStatFunc := func() {
		defer func() {
			if re := recover(); re != nil {
				buffer := debug.Stack()
				log.Error("SaveStatFunc")
				log.Error(re)
				log.Error(string(buffer))
			}
		}()
		stat := this.statMap.Get()
		if v, ok := stat[constDefine.CONST_STAT_FILE_COUNT_KEY]; ok {
			switch v.(type) {
			case int64, int32, int, float64, float32:
				if v.(int64) >= 0 {
					if data, err := json.Marshal(stat); err != nil {
						log.Error(err)
					} else {
						this.util.WriteBinFile(STAT_FILE_NAME, data)
					}
				}
			}
		}
	}
	SaveStatFunc()
}
func (this *Server) RemoveKeyFromLevelDB(key string, db *leveldb.DB) error {
	var (
		err error
	)
	err = db.Delete([]byte(key), nil)
	return err
}
func (this *Server) IsPeer(r *http.Request) bool {
	var (
		ip    string
		peer  string
		bflag bool
		cidr  *net.IPNet
		err   error
	)
	IsPublicIP := func(IP net.IP) bool {
		if IP.IsLoopback() || IP.IsLinkLocalMulticast() || IP.IsLinkLocalUnicast() {
			return false
		}
		if ip4 := IP.To4(); ip4 != nil {
			switch true {
			case ip4[0] == 10:
				return false
			case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
				return false
			case ip4[0] == 192 && ip4[1] == 168:
				return false
			default:
				return true
			}
		}
		return false
	}
	//return true
	ip = this.util.GetClientIp(r)
	if this.util.Contains("0.0.0.0", Config().AdminIps) {
		if IsPublicIP(net.ParseIP(ip)) {
			return false
		}
		return true
	}
	if this.util.Contains(ip, Config().AdminIps) {
		return true
	}
	for _, v := range Config().AdminIps {
		if strings.Contains(v, "/") {
			if _, cidr, err = net.ParseCIDR(v); err != nil {
				log.Error(err)
				return false
			}
			if cidr.Contains(net.ParseIP(ip)) {
				return true
			}
		}
	}
	realIp := os.Getenv("GFS_IP")
	if realIp == "" {
		realIp = this.util.GetPulicIP()
	}
	if ip == "127.0.0.1" || ip == realIp {
		return true
	}
	ip = "http://" + ip
	bflag = false
	for _, peer = range Config().Peers {
		if strings.HasPrefix(peer, ip) {
			bflag = true
			break
		}
	}
	return bflag
}
func (this *Server) GetClusterNotPermitMessage(r *http.Request) string {
	var (
		message string
	)
	message = fmt.Sprintf(constDefine.CONST_MESSAGE_CLUSTER_IP, this.util.GetClientIp(r))
	return message
}
func (this *Server) GetMd5sForWeb(w http.ResponseWriter, r *http.Request) {
	var (
		date   string
		err    error
		result mapset.Set
		lines  []string
		md5s   []interface{}
	)
	if !this.IsPeer(r) {
		w.Write([]byte(this.GetClusterNotPermitMessage(r)))
		return
	}
	date = r.FormValue("date")
	if result, err = this.GetMd5sByDate(date, constDefine.CONST_FILE_Md5_FILE_NAME); err != nil {
		log.Error(err)
		return
	}
	md5s = result.ToSlice()
	for _, line := range md5s {
		if line != nil && line != "" {
			lines = append(lines, line.(string))
		}
	}
	w.Write([]byte(strings.Join(lines, ",")))
}
func (this *Server) GetMd5File(w http.ResponseWriter, r *http.Request) {
	var (
		date  string
		fpath string
		data  []byte
		err   error
	)
	if !this.IsPeer(r) {
		return
	}
	fpath = DATA_DIR + "/" + date + "/" + constDefine.CONST_FILE_Md5_FILE_NAME
	if !this.util.FileExists(fpath) {
		w.WriteHeader(404)
		return
	}
	if data, err = ioutil.ReadFile(fpath); err != nil {
		w.WriteHeader(500)
		return
	}
	w.Write(data)
}
func (this *Server) GetMd5sMapByDate(date string, filename string) (*goutil.CommonMap, error) {
	var (
		err     error
		result  *goutil.CommonMap
		fpath   string
		content string
		lines   []string
		line    string
		cols    []string
		data    []byte
	)
	result = goutil.NewCommonMap(0)
	if filename == "" {
		fpath = DATA_DIR + "/" + date + "/" + constDefine.CONST_FILE_Md5_FILE_NAME
	} else {
		fpath = DATA_DIR + "/" + date + "/" + filename
	}
	if !this.util.FileExists(fpath) {
		return result, errors.New(fmt.Sprintf("fpath %s not found", fpath))
	}
	if data, err = ioutil.ReadFile(fpath); err != nil {
		return result, err
	}
	content = string(data)
	lines = strings.Split(content, "\n")
	for _, line = range lines {
		cols = strings.Split(line, "|")
		if len(cols) > 2 {
			if _, err = strconv.ParseInt(cols[1], 10, 64); err != nil {
				continue
			}
			result.Add(cols[0])
		}
	}
	return result, nil
}
func (this *Server) GetMd5sByDate(date string, filename string) (mapset.Set, error) {
	var (
		keyPrefix string
		md5set    mapset.Set
		keys      []string
	)
	md5set = mapset.NewSet()
	keyPrefix = "%s_%s_"
	keyPrefix = fmt.Sprintf(keyPrefix, date, filename)
	iter := Singleton.logDB.NewIterator(util.BytesPrefix([]byte(keyPrefix)), nil)
	for iter.Next() {
		keys = strings.Split(string(iter.Key()), "_")
		if len(keys) >= 3 {
			md5set.Add(keys[2])
		}
	}
	iter.Release()
	return md5set, nil
}
func (this *Server) SyncFileInfo(w http.ResponseWriter, r *http.Request) {
	var (
		err         error
		fileInfo    model.FileInfo
		fileInfoStr string
		filename    string
	)
	r.ParseForm()
	fileInfoStr = r.FormValue("fileInfo")
	if !this.IsPeer(r) {
		log.Info("isn't peer fileInfo:", fileInfo)
		return
	}
	if err = json.Unmarshal([]byte(fileInfoStr), &fileInfo); err != nil {
		w.Write([]byte(this.GetClusterNotPermitMessage(r)))
		log.Error(err)
		return
	}
	if fileInfo.OffSet == -2 {
		// optimize migrate
		this.SaveFileInfoToLevelDB(fileInfo.Md5, &fileInfo, this.ldb)
	} else {
		this.SaveFileMd5Log(&fileInfo, constDefine.CONST_Md5_QUEUE_FILE_NAME)
	}
	this.AppendToDownloadQueue(&fileInfo)
	filename = fileInfo.Name
	if fileInfo.ReName != "" {
		filename = fileInfo.ReName
	}
	p := strings.Replace(fileInfo.Path, STORE_DIR+"/", "", 1)
	downloadUrl := fmt.Sprintf("http://%s/%s", r.Host, Config().Group+"/"+p+"/"+filename)
	log.Info("SyncFileInfo: ", downloadUrl)
	w.Write([]byte(downloadUrl))
}
func (this *Server) GetFileInfo(w http.ResponseWriter, r *http.Request) {
	var (
		fpath    string
		md5sum   string
		fileInfo *model.FileInfo
		err      error
		result   model.JsonResult
	)
	md5sum = r.FormValue("md5")
	fpath = r.FormValue("path")
	result.Status = "fail"
	if !this.IsPeer(r) {
		w.Write([]byte(this.GetClusterNotPermitMessage(r)))
		return
	}
	md5sum = r.FormValue("md5")
	if fpath != "" {
		fpath = strings.Replace(fpath, "/"+Config().Group+"/", constDefine.STORE_DIR_NAME+"/", 1)
		md5sum = this.util.MD5(fpath)
	}
	if fileInfo, err = this.GetFileInfoFromLevelDB(md5sum); err != nil {
		log.Error(err)
		result.Message = err.Error()
		w.Write([]byte(this.util.JsonEncodePretty(result)))
		return
	}
	result.Status = "ok"
	result.Data = fileInfo
	w.Write([]byte(this.util.JsonEncodePretty(result)))
	return
}
func (this *Server) RemoveFile(w http.ResponseWriter, r *http.Request) {
	var (
		err      error
		md5sum   string
		fileInfo *model.FileInfo
		fpath    string
		delUrl   string
		result   model.JsonResult
		inner    string
		name     string
	)
	_ = delUrl
	_ = inner
	r.ParseForm()
	md5sum = r.FormValue("md5")
	fpath = r.FormValue("path")
	inner = r.FormValue("inner")
	result.Status = "fail"
	if !this.IsPeer(r) {
		w.Write([]byte(this.GetClusterNotPermitMessage(r)))
		return
	}
	if Config().AuthUrl != "" && !this.CheckAuth(w, r) {
		this.NotPermit(w, r)
		return
	}
	if fpath != "" && md5sum == "" {
		fpath = strings.Replace(fpath, "/"+Config().Group+"/", constDefine.STORE_DIR_NAME+"/", 1)
		md5sum = this.util.MD5(fpath)
	}
	if inner != "1" {
		for _, peer := range Config().Peers {
			delFile := func(peer string, md5sum string, fileInfo *model.FileInfo) {
				delUrl = fmt.Sprintf("%s%s", peer, this.getRequestURI("delete"))
				req := httplib.Post(delUrl)
				req.Param("md5", md5sum)
				req.Param("inner", "1")
				req.SetTimeout(time.Second*5, time.Second*10)
				if _, err = req.String(); err != nil {
					log.Error(err)
				}
			}
			go delFile(peer, md5sum, fileInfo)
		}
	}
	if len(md5sum) < 32 {
		result.Message = "md5 unvalid"
		w.Write([]byte(this.util.JsonEncodePretty(result)))
		return
	}
	if fileInfo, err = this.GetFileInfoFromLevelDB(md5sum); err != nil {
		result.Message = err.Error()
		w.Write([]byte(this.util.JsonEncodePretty(result)))
		return
	}
	if fileInfo.OffSet >= 0 {
		result.Message = "small file delete not support"
		w.Write([]byte(this.util.JsonEncodePretty(result)))
		return
	}
	name = fileInfo.Name
	if fileInfo.ReName != "" {
		name = fileInfo.ReName
	}
	fpath = fileInfo.Path + "/" + name
	if fileInfo.Path != "" && this.util.FileExists(DOCKER_DIR+fpath) {
		this.SaveFileMd5Log(fileInfo, constDefine.CONST_REMOME_Md5_FILE_NAME)
		if err = os.Remove(DOCKER_DIR + fpath); err != nil {
			result.Message = err.Error()
			w.Write([]byte(this.util.JsonEncodePretty(result)))
			return
		} else {
			result.Message = "remove success"
			result.Status = "ok"
			w.Write([]byte(this.util.JsonEncodePretty(result)))
			return
		}
	}
	result.Message = "fail remove"
	w.Write([]byte(this.util.JsonEncodePretty(result)))
}
func (this *Server) getRequestURI(action string) string {
	var (
		uri string
	)
	if Config().SupportGroupManage {
		uri = "/" + Config().Group + "/" + action
	} else {
		uri = "/" + action
	}
	return uri
}
func (this *Server) BuildFileResult(fileInfo *model.FileInfo, r *http.Request) model.FileResult {
	var (
		outname     string
		fileResult  model.FileResult
		p           string
		downloadUrl string
		domain      string
		host        string
		protocol    string
	)
	if Config().EnableHttps {
		protocol = "https"
	} else {
		protocol = "http"
	}
	host = strings.Replace(Config().Host, "http://", "", -1)
	if r != nil {
		host = r.Host
	}
	if !strings.HasPrefix(Config().DownloadDomain, "http") {
		if Config().DownloadDomain == "" {
			Config().DownloadDomain = fmt.Sprintf("%s://%s", protocol, host)
		} else {
			Config().DownloadDomain = fmt.Sprintf("%s://%s", protocol, Config().DownloadDomain)
		}
	}
	if Config().DownloadDomain != "" {
		domain = Config().DownloadDomain
	} else {
		domain = fmt.Sprintf("%s://%s", protocol, host)
	}
	outname = fileInfo.Name
	if fileInfo.ReName != "" {
		outname = fileInfo.ReName
	}
	p = strings.Replace(fileInfo.Path, constDefine.STORE_DIR_NAME+"/", "", 1)
	if Config().SupportGroupManage {
		p = Config().Group + "/" + p + "/" + outname
	} else {
		p = p + "/" + outname
	}
	downloadUrl = fmt.Sprintf("%s://%s/%s", protocol, host, p)
	if Config().DownloadDomain != "" {
		downloadUrl = fmt.Sprintf("%s/%s", Config().DownloadDomain, p)
	}
	fileResult.Url = downloadUrl
	fileResult.Md5 = fileInfo.Md5
	fileResult.Path = "/" + p
	fileResult.Domain = domain
	fileResult.Scene = fileInfo.Scene
	fileResult.Size = fileInfo.Size
	fileResult.ModTime = fileInfo.TimeStamp
	// Just for Compatibility
	fileResult.Src = fileResult.Path
	fileResult.Scenes = fileInfo.Scene
	return fileResult
}
func (this *Server) RegisterExit() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		for s := range c {
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				this.ldb.Close()
				log.Info("Exit", s)
				os.Exit(1)
			}
		}
	}()
}
func (this *Server) AppendToQueue(fileInfo *model.FileInfo) {
	for (len(this.queueToPeers) + constDefine.CONST_QUEUE_SIZE/10) > constDefine.CONST_QUEUE_SIZE {
		time.Sleep(time.Millisecond * 50)
	}
	this.queueToPeers <- *fileInfo
}
func (this *Server) AppendToDownloadQueue(fileInfo *model.FileInfo) {
	for (len(this.queueFromPeers) + constDefine.CONST_QUEUE_SIZE/10) > constDefine.CONST_QUEUE_SIZE {
		time.Sleep(time.Millisecond * 50)
	}
	this.queueFromPeers <- *fileInfo
}
func (this *Server) ConsumerDownLoad() {
	ConsumerFunc := func() {
		for {
			fileInfo := <-this.queueFromPeers
			if len(fileInfo.Peers) <= 0 {
				log.Warn("Peer is null", fileInfo)
				continue
			}
			for _, peer := range fileInfo.Peers {
				if strings.Contains(peer, "127.0.0.1") {
					log.Warn("sync error with 127.0.0.1", fileInfo)
					continue
				}
				if peer != this.host {
					this.DownloadFromPeer(peer, &fileInfo)
					break
				}
			}
		}
	}
	for i := 0; i < Config().SyncWorker; i++ {
		go ConsumerFunc()
	}
}
func (this *Server) RemoveDownloading() {
	RemoveDownloadFunc := func() {
		for {
			iter := this.ldb.NewIterator(util.BytesPrefix([]byte("downloading_")), nil)
			for iter.Next() {
				key := iter.Key()
				keys := strings.Split(string(key), "_")
				if len(keys) == 3 {
					if t, err := strconv.ParseInt(keys[1], 10, 64); err == nil && time.Now().Unix()-t > 60*10 {
						os.Remove(DOCKER_DIR + keys[2])
					}
				}
			}
			iter.Release()
			time.Sleep(time.Minute * 3)
		}
	}
	go RemoveDownloadFunc()
}
func (this *Server) ConsumerLog() {
	go func() {
		var (
			fileLog *model.FileLog
		)
		for {
			fileLog = <-this.queueFileLog
			this.saveFileMd5Log(fileLog.FileInfo, fileLog.FileName)
		}
	}()
}
func (this *Server) LoadSearchDict() {
	go func() {
		log.Info("Load search dict ....")
		f, err := os.Open(SEARCH_FILE_NAME)
		if err != nil {
			log.Error(err)
			return
		}
		defer f.Close()
		r := bufio.NewReader(f)
		for {
			line, isprefix, err := r.ReadLine()
			for isprefix && err == nil {
				kvs := strings.Split(string(line), "\t")
				if len(kvs) == 2 {
					this.searchMap.Put(kvs[0], kvs[1])
				}
			}
		}
		log.Info("finish load search dict")
	}()
}
func (this *Server) SaveSearchDict() {
	var (
		err        error
		fp         *os.File
		searchDict map[string]interface{}
		k          string
		v          interface{}
	)
	this.lockMap.LockKey(SEARCH_FILE_NAME)
	defer this.lockMap.UnLockKey(SEARCH_FILE_NAME)
	searchDict = this.searchMap.Get()
	fp, err = os.OpenFile(SEARCH_FILE_NAME, os.O_RDWR, 0755)
	if err != nil {
		log.Error(err)
		return
	}
	defer fp.Close()
	for k, v = range searchDict {
		fp.WriteString(fmt.Sprintf("%s\t%s", k, v.(string)))
	}
}
func (this *Server) ConsumerPostToPeer() {
	ConsumerFunc := func() {
		for {
			fileInfo := <-this.queueToPeers
			this.postFileToPeer(&fileInfo)
		}
	}
	for i := 0; i < Config().SyncWorker; i++ {
		go ConsumerFunc()
	}
}
func (this *Server) CleanLogLevelDBByDate(date string, filename string) {
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			log.Error("CleanLogLevelDBByDate")
			log.Error(re)
			log.Error(string(buffer))
		}
	}()
	var (
		err       error
		keyPrefix string
		keys      mapset.Set
	)
	keys = mapset.NewSet()
	keyPrefix = "%s_%s_"
	keyPrefix = fmt.Sprintf(keyPrefix, date, filename)
	iter := Singleton.logDB.NewIterator(util.BytesPrefix([]byte(keyPrefix)), nil)
	for iter.Next() {
		keys.Add(string(iter.Value()))
	}
	iter.Release()
	for key := range keys.Iter() {
		err = this.RemoveKeyFromLevelDB(key.(string), this.logDB)
		if err != nil {
			log.Error(err)
		}
	}
}
func (this *Server) LoadQueueSendToPeer() {
	if queue, err := this.LoadFileInfoByDate(this.util.GetToDay(), constDefine.CONST_Md5_QUEUE_FILE_NAME); err != nil {
		log.Error(err)
	} else {
		for fileInfo := range queue.Iter() {
			//this.queueFromPeers <- *fileInfo.(*FileInfo)
			this.AppendToDownloadQueue(fileInfo.(*model.FileInfo))
		}
	}
}
func (this *Server) CheckClusterStatus() {
	check := func() {
		defer func() {
			if re := recover(); re != nil {
				buffer := debug.Stack()
				log.Error("CheckClusterStatus")
				log.Error(re)
				log.Error(string(buffer))
			}
		}()
		var (
			status  model.JsonResult
			err     error
			subject string
			body    string
			req     *httplib.BeegoHTTPRequest
		)
		for _, peer := range Config().Peers {
			req = httplib.Get(fmt.Sprintf("%s%s", peer, this.getRequestURI("status")))
			req.SetTimeout(time.Second*5, time.Second*5)
			err = req.ToJSON(&status)
			if err != nil || status.Status != "ok" {
				if err != nil {
					body = fmt.Sprintf("%s\nserver:%s\nerror:\n%s", subject, peer, err.Error())
				} else {
					body = fmt.Sprintf("%s\nserver:%s\n", subject, peer)
				}
				log.Error(body)
			}
		}
	}
	go func() {
		for {
			check()
			time.Sleep(time.Minute * 10)
			check()
		}
	}()
}
func (this *Server) RepairFileInfo(w http.ResponseWriter, r *http.Request) {
	var (
		result model.JsonResult
	)
	if !this.IsPeer(r) {
		w.Write([]byte(this.GetClusterNotPermitMessage(r)))
		return
	}
	if !Config().EnableMigrate {
		w.Write([]byte("please set enable_migrate=true"))
		return
	}
	result.Status = "ok"
	result.Message = "repair job start,don't try again,very danger "
	go this.RepairFileInfoFromFile()
	w.Write([]byte(this.util.JsonEncodePretty(result)))
}
func (this *Server) Reload(w http.ResponseWriter, r *http.Request) {
	var (
		err     error
		data    []byte
		cfg     configRepo.GloablConfig
		action  string
		cfgjson string
		result  model.JsonResult
	)
	result.Status = "fail"
	r.ParseForm()
	if !this.IsPeer(r) {
		w.Write([]byte(this.GetClusterNotPermitMessage(r)))
		return
	}
	cfgjson = r.FormValue("cfg")
	action = r.FormValue("action")
	_ = cfgjson
	if action == "get" {
		result.Data = Config()
		result.Status = "ok"
		w.Write([]byte(this.util.JsonEncodePretty(result)))
		return
	}
	if action == "set" {
		if cfgjson == "" {
			result.Message = "(error)parameter cfg(json) require"
			w.Write([]byte(this.util.JsonEncodePretty(result)))
			return
		}
		if err = json.Unmarshal([]byte(cfgjson), &cfg); err != nil {
			log.Error(err)
			result.Message = err.Error()
			w.Write([]byte(this.util.JsonEncodePretty(result)))
			return
		}
		result.Status = "ok"
		cfgjson = this.util.JsonEncodePretty(cfg)
		this.util.WriteFile(CONF_FILE_NAME, cfgjson)
		w.Write([]byte(this.util.JsonEncodePretty(result)))
		return
	}
	if action == "reload" {
		if data, err = ioutil.ReadFile(CONF_FILE_NAME); err != nil {
			result.Message = err.Error()
			w.Write([]byte(this.util.JsonEncodePretty(result)))
			return
		}
		if err = json.Unmarshal(data, &cfg); err != nil {
			result.Message = err.Error()
			w.Write([]byte(this.util.JsonEncodePretty(result)))
			return
		}
		ParseConfig(CONF_FILE_NAME)
		this.initComponent(true)
		result.Status = "ok"
		w.Write([]byte(this.util.JsonEncodePretty(result)))
		return
	}
	if action == "" {
		w.Write([]byte("(error)action support set(json) get reload"))
	}
}
func (this *Server) RemoveEmptyDir(w http.ResponseWriter, r *http.Request) {
	var (
		result model.JsonResult
	)
	result.Status = "ok"
	if this.IsPeer(r) {
		go this.util.RemoveEmptyDir(DATA_DIR)
		go this.util.RemoveEmptyDir(STORE_DIR)
		result.Message = "clean job start ..,don't try again!!!"
		w.Write([]byte(this.util.JsonEncodePretty(result)))
	} else {
		result.Message = this.GetClusterNotPermitMessage(r)
		w.Write([]byte(this.util.JsonEncodePretty(result)))
	}
}

func (this *Server) ReceiveMd5s(w http.ResponseWriter, r *http.Request) {
	var (
		err      error
		md5str   string
		fileInfo *model.FileInfo
		md5s     []string
	)
	if !this.IsPeer(r) {
		log.Warn(fmt.Sprintf("ReceiveMd5s %s", this.util.GetClientIp(r)))
		w.Write([]byte(this.GetClusterNotPermitMessage(r)))
		return
	}
	r.ParseForm()
	md5str = r.FormValue("md5s")
	md5s = strings.Split(md5str, ",")
	AppendFunc := func(md5s []string) {
		for _, m := range md5s {
			if m != "" {
				if fileInfo, err = this.GetFileInfoFromLevelDB(m); err != nil {
					log.Error(err)
					continue
				}
				this.AppendToQueue(fileInfo)
			}
		}
	}
	go AppendFunc(md5s)
}
func (this *Server) ListDir(w http.ResponseWriter, r *http.Request) {
	var (
		result      model.JsonResult
		dir         string
		filesInfo   []os.FileInfo
		err         error
		filesResult []model.FileInfoResult
		tmpDir      string
	)
	if !this.IsPeer(r) {
		result.Message = this.GetClusterNotPermitMessage(r)
		w.Write([]byte(this.util.JsonEncodePretty(result)))
		return
	}
	dir = r.FormValue("dir")
	//if dir == "" {
	//	result.Message = "dir can't null"
	//	w.Write([]byte(this.util.JsonEncodePretty(result)))
	//	return
	//}
	dir = strings.Replace(dir, ".", "", -1)
	if tmpDir, err = os.Readlink(dir); err == nil {
		dir = tmpDir
	}
	filesInfo, err = ioutil.ReadDir(DOCKER_DIR + constDefine.STORE_DIR_NAME + "/" + dir)
	if err != nil {
		log.Error(err)
		result.Message = err.Error()
		w.Write([]byte(this.util.JsonEncodePretty(result)))
		return
	}
	for _, f := range filesInfo {
		fi := model.FileInfoResult{
			Name:    f.Name(),
			Size:    f.Size(),
			IsDir:   f.IsDir(),
			ModTime: f.ModTime().Unix(),
			Path:    dir,
			Md5:     this.util.MD5(strings.Replace(constDefine.STORE_DIR_NAME+"/"+dir+"/"+f.Name(), "//", "/", -1)),
		}
		filesResult = append(filesResult, fi)
	}
	result.Status = "ok"
	result.Data = filesResult
	w.Write([]byte(this.util.JsonEncodePretty(result)))
	return
}
func (this *Server) Report(w http.ResponseWriter, r *http.Request) {
	var (
		reportFileName string
		result         model.JsonResult
		html           string
	)
	result.Status = "ok"
	r.ParseForm()
	if this.IsPeer(r) {
		reportFileName = STATIC_DIR + "/report.html"
		if this.util.IsExist(reportFileName) {
			if data, err := this.util.ReadBinFile(reportFileName); err != nil {
				log.Error(err)
				result.Message = err.Error()
				w.Write([]byte(this.util.JsonEncodePretty(result)))
				return
			} else {
				html = string(data)
				if Config().SupportGroupManage {
					html = strings.Replace(html, "{group}", "/"+Config().Group, 1)
				} else {
					html = strings.Replace(html, "{group}", "", 1)
				}
				w.Write([]byte(html))
				return
			}
		} else {
			w.Write([]byte(fmt.Sprintf("%s is not found", reportFileName)))
		}
	} else {
		w.Write([]byte(this.GetClusterNotPermitMessage(r)))
	}
}
func (this *Server) Repair(w http.ResponseWriter, r *http.Request) {
	var (
		force       string
		forceRepair bool
		result      model.JsonResult
	)
	result.Status = "ok"
	r.ParseForm()
	force = r.FormValue("force")
	if force == "1" {
		forceRepair = true
	}
	if this.IsPeer(r) {
		go this.AutoRepair(forceRepair)
		result.Message = "repair job start..."
		w.Write([]byte(this.util.JsonEncodePretty(result)))
	} else {
		result.Message = this.GetClusterNotPermitMessage(r)
		w.Write([]byte(this.util.JsonEncodePretty(result)))
	}

}
func (this *Server) Index(w http.ResponseWriter, r *http.Request) {
	var (
		uploadUrl    string
		uploadBigUrl string
		uppy         string
	)
	uploadUrl = "/upload"
	uploadBigUrl = constDefine.CONST_BIG_UPLOAD_PATH_SUFFIX
	if Config().EnableWebUpload {
		if Config().SupportGroupManage {
			uploadUrl = fmt.Sprintf("/%s/upload", Config().Group)
			uploadBigUrl = fmt.Sprintf("/%s%s", Config().Group, constDefine.CONST_BIG_UPLOAD_PATH_SUFFIX)
		}
		uppy = `<html>
			  
			  <head>
				<meta charset="utf-8" />
				<title>gfs</title>
				<style>form { bargin } .form-line { display:block;height: 30px;margin:8px; } #stdUpload {background: #fafafa;border-radius: 10px;width: 745px; }</style>
				<link href="https://transloadit.edgly.net/releases/uppy/v0.30.0/dist/uppy.min.css" rel="stylesheet"></head>
			  
			  <body>
                <div>标准上传(强列建议使用这种方式)</div>
				<div id="stdUpload">
				  
				  <form action="%s" method="post" enctype="multipart/form-data">
					<span class="form-line">文件(file):
					  <input type="file" id="file" name="file" /></span>
					<span class="form-line">场景(scene):
					  <input type="text" id="scene" name="scene" value="%s" /></span>
					<span class="form-line">文件名(filename):
					  <input type="text" id="filename" name="filename" value="" /></span>
					<span class="form-line">输出(output):
					  <input type="text" id="output" name="output" value="json2" title="json|text|json2" /></span>
					<span class="form-line">自定义路径(path):
					  <input type="text" id="path" name="path" value="" /></span>
	              <span class="form-line">google认证码(code):
					  <input type="text" id="code" name="code" value="" /></span>
					 <span class="form-line">自定义认证(auth_token):
					  <input type="text" id="auth_token" name="auth_token" value="" /></span>
					<input type="submit" name="submit" value="upload" />
                </form>
				</div>
                 <div>断点续传（如果文件很大时可以考虑）</div>
				<div>
				 
				  <div id="drag-drop-area"></div>
				  <script src="https://transloadit.edgly.net/releases/uppy/v0.30.0/dist/uppy.min.js"></script>
				  <script>var uppy = Uppy.Core().use(Uppy.Dashboard, {
					  inline: true,
					  target: '#drag-drop-area'
					}).use(Uppy.Tus, {
					  endpoint: '%s'
					})
					uppy.on('complete', (result) => {
					 // console.log(result) console.log('Upload complete! We’ve uploaded these files:', result.successful)
					})
					//uppy.setMeta({ auth_token: '9ee60e59-cb0f-4578-aaba-29b9fc2919ca',callback_url:'http://127.0.0.1/callback' ,filename:'自定义文件名','path':'自定义path',scene:'自定义场景' })//这里是传递上传的认证参数,callback_url参数中 id为文件的ID,info 文转的基本信息json
					uppy.setMeta({ auth_token: '9ee60e59-cb0f-4578-aaba-29b9fc2919ca',callback_url:'http://127.0.0.1/callback'})//自定义参数与普通上传类似（虽然支持自定义，建议不要自定义，海量文件情况下，自定义很可能给自已给埋坑）
                </script>
				</div>
			  </body>
			</html>`
		uppyFileName := STATIC_DIR + "/uppy.html"
		if this.util.IsExist(uppyFileName) {
			if data, err := this.util.ReadBinFile(uppyFileName); err != nil {
				log.Error(err)
			} else {
				uppy = string(data)
			}
		} else {
			this.util.WriteFile(uppyFileName, uppy)
		}
		fmt.Fprintf(w,
			fmt.Sprintf(uppy, uploadUrl, Config().DefaultScene, uploadBigUrl))
	} else {
		w.Write([]byte("web upload deny"))
	}
}

func (this *Server) initTus() {
	var (
		err     error
		fileLog *os.File
		bigDir  string
	)
	BIG_DIR := STORE_DIR + "/_big/" + Config().PeerId
	os.MkdirAll(BIG_DIR, 0775)
	os.MkdirAll(LOG_DIR, 0775)
	store := filestore.FileStore{
		Path: BIG_DIR,
	}
	if fileLog, err = os.OpenFile(LOG_DIR+"/tusd.log", os.O_CREATE|os.O_RDWR, 0666); err != nil {
		log.Error(err)
		panic("initTus")
	}
	go func() {
		for {
			if fi, err := fileLog.Stat(); err != nil {
				log.Error(err)
			} else {
				if fi.Size() > 1024*1024*500 {
					//500M
					this.util.CopyFile(LOG_DIR+"/tusd.log", LOG_DIR+"/tusd.log.2")
					fileLog.Seek(0, 0)
					fileLog.Truncate(0)
					fileLog.Seek(0, 2)
				}
			}
			time.Sleep(time.Second * 30)
		}
	}()
	l := slog.New(fileLog, "[tusd] ", slog.LstdFlags)
	bigDir = constDefine.CONST_BIG_UPLOAD_PATH_SUFFIX
	if Config().SupportGroupManage {
		bigDir = fmt.Sprintf("/%s%s", Config().Group, constDefine.CONST_BIG_UPLOAD_PATH_SUFFIX)
	}
	composer := tusd.NewStoreComposer()
	// support raw tus upload and download
	store.GetReaderExt = func(id string) (io.Reader, error) {
		var (
			offset int64
			err    error
			length int
			buffer []byte
			fi     *model.FileInfo
			fn     string
		)
		if fi, err = this.GetFileInfoFromLevelDB(id); err != nil {
			log.Error(err)
			return nil, err
		} else {
			if Config().AuthUrl != "" {
				fileResult := this.util.JsonEncodePretty(this.BuildFileResult(fi, nil))
				bufferReader := bytes.NewBuffer([]byte(fileResult))
				return bufferReader, nil
			}
			fn = fi.Name
			if fi.ReName != "" {
				fn = fi.ReName
			}
			fp := DOCKER_DIR + fi.Path + "/" + fn
			if this.util.FileExists(fp) {
				log.Info(fmt.Sprintf("download:%s", fp))
				return os.Open(fp)
			}
			ps := strings.Split(fp, ",")
			if len(ps) > 2 && this.util.FileExists(ps[0]) {
				if length, err = strconv.Atoi(ps[2]); err != nil {
					return nil, err
				}
				if offset, err = strconv.ParseInt(ps[1], 10, 64); err != nil {
					return nil, err
				}
				if buffer, err = this.util.ReadFileByOffSet(ps[0], offset, length); err != nil {
					return nil, err
				}
				if buffer[0] == '1' {
					bufferReader := bytes.NewBuffer(buffer[1:])
					return bufferReader, nil
				} else {
					msg := "data no sync"
					log.Error(msg)
					return nil, errors.New(msg)
				}
			}
			return nil, errors.New(fmt.Sprintf("%s not found", fp))
		}
	}
	store.UseIn(composer)
	SetupPreHooks := func(composer *tusd.StoreComposer) {
		composer.UseCore(HookDataStore{
			DataStore: composer.Core,
		})
	}
	SetupPreHooks(composer)
	handler, err := tusd.NewHandler(tusd.Config{
		Logger:                  l,
		BasePath:                bigDir,
		StoreComposer:           composer,
		NotifyCompleteUploads:   true,
		RespectForwardedHeaders: true,
	})
	notify := func(handler *tusd.Handler) {
		for {
			select {
			case info := <-handler.CompleteUploads:
				log.Info("CompleteUploads", info)
				name := ""
				pathCustom := ""
				scene := Config().DefaultScene
				if v, ok := info.MetaData["filename"]; ok {
					name = v
				}
				if v, ok := info.MetaData["scene"]; ok {
					scene = v
				}
				if v, ok := info.MetaData["path"]; ok {
					pathCustom = v
				}
				var err error
				md5sum := ""
				oldFullPath := BIG_DIR + "/" + info.ID + ".bin"
				infoFullPath := BIG_DIR + "/" + info.ID + ".info"
				if md5sum, err = this.util.GetFileSumByName(oldFullPath, Config().FileSumArithmetic); err != nil {
					log.Error(err)
					continue
				}
				ext := path.Ext(name)
				filename := md5sum + ext
				if name != "" {
					filename = name
				}
				if Config().RenameFile {
					filename = md5sum + ext
				}
				timeStamp := time.Now().Unix()
				fpath := time.Now().Format("/20060102/15/04/")
				if pathCustom != "" {
					fpath = "/" + strings.Replace(pathCustom, ".", "", -1) + "/"
				}
				newFullPath := STORE_DIR + "/" + scene + fpath + Config().PeerId + "/" + filename
				if pathCustom != "" {
					newFullPath = STORE_DIR + "/" + scene + fpath + filename
				}
				if fi, err := this.GetFileInfoFromLevelDB(md5sum); err != nil {
					log.Error(err)
				} else {
					tpath := this.GetFilePathByInfo(fi, true)
					if fi.Md5 != "" && this.util.FileExists(tpath) {
						if _, err := this.SaveFileInfoToLevelDB(info.ID, fi, this.ldb); err != nil {
							log.Error(err)
						}
						log.Info(fmt.Sprintf("file is found md5:%s", fi.Md5))
						log.Info("remove file:", oldFullPath)
						log.Info("remove file:", infoFullPath)
						os.Remove(oldFullPath)
						os.Remove(infoFullPath)
						continue
					}
				}
				fpath2 := ""
				fpath2 = constDefine.STORE_DIR_NAME + "/" + Config().DefaultScene + fpath + Config().PeerId
				if pathCustom != "" {
					fpath2 = constDefine.STORE_DIR_NAME + "/" + Config().DefaultScene + fpath
					fpath2 = strings.TrimRight(fpath2, "/")
				}

				os.MkdirAll(DOCKER_DIR+fpath2, 0775)
				fileInfo := &model.FileInfo{
					Name:      name,
					Path:      fpath2,
					ReName:    filename,
					Size:      info.Size,
					TimeStamp: timeStamp,
					Md5:       md5sum,
					Peers:     []string{this.host},
					OffSet:    -1,
				}
				if err = os.Rename(oldFullPath, newFullPath); err != nil {
					log.Error(err)
					continue
				}
				log.Info(fileInfo)
				os.Remove(infoFullPath)
				if _, err = this.SaveFileInfoToLevelDB(info.ID, fileInfo, this.ldb); err != nil {
					//assosiate file id
					log.Error(err)
				}
				this.SaveFileMd5Log(fileInfo, constDefine.CONST_FILE_Md5_FILE_NAME)
				go this.postFileToPeer(fileInfo)
				callBack := func(info tusd.FileInfo, fileInfo *model.FileInfo) {
					if callback_url, ok := info.MetaData["callback_url"]; ok {
						req := httplib.Post(callback_url)
						req.SetTimeout(time.Second*10, time.Second*10)
						req.Param("info", Singleton.util.JsonEncodePretty(fileInfo))
						req.Param("id", info.ID)
						if _, err := req.String(); err != nil {
							log.Error(err)
						}
					}
				}
				go callBack(info, fileInfo)
			}
		}
	}
	go notify(handler)
	if err != nil {
		log.Error(err)
	}
	http.Handle(bigDir, http.StripPrefix(bigDir, handler))
}

func (this *Server) initComponent(isReload bool) {
	var (
		ip string
	)
	if ip = os.Getenv("GFS_IP"); ip == "" {
		ip = this.util.GetPulicIP()
	}
	if Config().Host == "" {
		if len(strings.Split(Config().Addr, ":")) == 2 {
			Singleton.host = fmt.Sprintf("http://%s:%s", ip, strings.Split(Config().Addr, ":")[1])
			Config().Host = Singleton.host
		}
	} else {
		if strings.HasPrefix(Config().Host, "http") {
			Singleton.host = Config().Host
		} else {
			Singleton.host = "http://" + Config().Host
		}
	}
	ex, _ := regexp.Compile("\\d+\\.\\d+\\.\\d+\\.\\d+")
	var peers []string
	for _, peer := range Config().Peers {
		if this.util.Contains(ip, ex.FindAllString(peer, -1)) ||
			this.util.Contains("127.0.0.1", ex.FindAllString(peer, -1)) {
			continue
		}
		if strings.HasPrefix(peer, "http") {
			peers = append(peers, peer)
		} else {
			peers = append(peers, "http://"+peer)
		}
	}
	Config().Peers = peers
	if !isReload {
		this.FormatStatInfo()
		if Config().EnableTus {
			this.initTus()
		}
	}
	for _, s := range Config().Scenes {
		kv := strings.Split(s, ":")
		if len(kv) == 2 {
			this.sceneMap.Put(kv[0], kv[1])
		}
	}
	if Config().ReadTimeout == 0 {
		Config().ReadTimeout = 60 * 10
	}
	if Config().WriteTimeout == 0 {
		Config().WriteTimeout = 60 * 10
	}
	if Config().SyncWorker == 0 {
		Config().SyncWorker = 200
	}
	if Config().UploadWorker == 0 {
		Config().UploadWorker = runtime.NumCPU() + 4
		if runtime.NumCPU() < 4 {
			Config().UploadWorker = 8
		}
	}
	if Config().UploadQueueSize == 0 {
		Config().UploadQueueSize = 200
	}
	if Config().RetryCount == 0 {
		Config().RetryCount = 3
	}
	if Config().SyncDelay == 0 {
		Config().SyncDelay = 60
	}
	if Config().WatchChanSize == 0 {
		Config().WatchChanSize = 100000
	}
}

func (this *Server) Start() {
	go func() {
		for {
			//重试同步失败的文件
			this.CheckFileAndSendToPeer(this.util.GetToDay(), constDefine.CONST_Md5_ERROR_FILE_NAME, false)
			//fmt.Println("CheckFileAndSendToPeer")
			time.Sleep(time.Second * time.Duration(Config().RefreshInterval))
			//this.util.RemoveEmptyDir(STORE_DIR)
		}
	}()
	go this.CleanAndBackUp()
	//定时检测节点状态
	go this.CheckClusterStatus()
	//获取要
	go this.LoadQueueSendToPeer()
	//发送到其他节点
	go this.ConsumerPostToPeer()
	go this.ConsumerLog()
	//下载
	go this.ConsumerDownLoad()
	//上传
	go this.ConsumerUpload()
	//下载超时一定时间的删除
	go this.RemoveDownloading()
	//go this.LoadSearchDict()
	if Config().EnableMigrate {
		go this.RepairFileInfoFromFile()
	}
	if Config().AutoRepair {
		go func() {
			for {
				time.Sleep(time.Minute * 3)
				this.AutoRepair(false)
				time.Sleep(time.Minute * 60)
			}
		}()
	}
	groupRoute := ""
	if Config().SupportGroupManage {
		groupRoute = "/" + Config().Group
	}
	go func() { // force free memory
		for {
			time.Sleep(time.Minute * 1)
			debug.FreeOSMemory()
		}
	}()
	uploadPage := "upload.html"
	if groupRoute == "" {
		http.HandleFunc(fmt.Sprintf("%s", "/"), this.Download)
		http.HandleFunc(fmt.Sprintf("/%s", uploadPage), this.Index)
	} else {
		http.HandleFunc(fmt.Sprintf("%s", "/"), this.Download)
		http.HandleFunc(fmt.Sprintf("%s", groupRoute), this.Download)
		http.HandleFunc(fmt.Sprintf("%s/%s", groupRoute, uploadPage), this.Index)
	}
	http.HandleFunc(fmt.Sprintf("%s/check_files_exist", groupRoute), this.CheckFilesExist)
	http.HandleFunc(fmt.Sprintf("%s/check_file_exist", groupRoute), this.CheckFileExist)
	http.HandleFunc(fmt.Sprintf("%s/upload", groupRoute), this.Upload)
	http.HandleFunc(fmt.Sprintf("%s/delete", groupRoute), this.RemoveFile)
	http.HandleFunc(fmt.Sprintf("%s/get_file_info", groupRoute), this.GetFileInfo)
	http.HandleFunc(fmt.Sprintf("%s/sync", groupRoute), this.Sync)
	http.HandleFunc(fmt.Sprintf("%s/stat", groupRoute), this.Stat)
	http.HandleFunc(fmt.Sprintf("%s/repair_stat", groupRoute), this.RepairStatWeb)
	http.HandleFunc(fmt.Sprintf("%s/status", groupRoute), this.Status)
	http.HandleFunc(fmt.Sprintf("%s/repair", groupRoute), this.Repair)
	http.HandleFunc(fmt.Sprintf("%s/report", groupRoute), this.Report)
	http.HandleFunc(fmt.Sprintf("%s/backup", groupRoute), this.BackUp)
	http.HandleFunc(fmt.Sprintf("%s/search", groupRoute), this.Search)
	http.HandleFunc(fmt.Sprintf("%s/list_dir", groupRoute), this.ListDir)
	http.HandleFunc(fmt.Sprintf("%s/remove_empty_dir", groupRoute), this.RemoveEmptyDir)
	http.HandleFunc(fmt.Sprintf("%s/repair_fileinfo", groupRoute), this.RepairFileInfo)
	http.HandleFunc(fmt.Sprintf("%s/reload", groupRoute), this.Reload)
	http.HandleFunc(fmt.Sprintf("%s/syncfile_info", groupRoute), this.SyncFileInfo)
	http.HandleFunc(fmt.Sprintf("%s/get_md5s_by_date", groupRoute), this.GetMd5sForWeb)
	http.HandleFunc(fmt.Sprintf("%s/receive_md5s", groupRoute), this.ReceiveMd5s)
	http.Handle(fmt.Sprintf("%s/static/", groupRoute), http.StripPrefix(fmt.Sprintf("%s/static/", groupRoute), http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/"+Config().Group+"/", this.Download)
	fmt.Println("Listen on " + Config().Addr)
	if Config().EnableHttps {
		err := http.ListenAndServeTLS(Config().Addr, SERVER_CRT_FILE_NAME, SERVER_KEY_FILE_NAME, new(HttpHandler))
		log.Error(err)
		fmt.Println(err)
	} else {
		srv := &http.Server{
			Addr:              Config().Addr,
			Handler:           new(HttpHandler),
			ReadTimeout:       time.Duration(Config().ReadTimeout) * time.Second,
			ReadHeaderTimeout: time.Duration(Config().ReadHeaderTimeout) * time.Second,
			WriteTimeout:      time.Duration(Config().WriteTimeout) * time.Second,
			IdleTimeout:       time.Duration(Config().IdleTimeout) * time.Second,
		}
		err := srv.ListenAndServe()
		log.Error(err)
		fmt.Println(err)
	}
}
