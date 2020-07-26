package core

import (
	"fmt"
	"github.com/astaxie/beego/httplib"
	mapset "github.com/deckarep/golang-set"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	log "github.com/sjqzhang/seelog"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/xukgo/gfs/constDefine"
	"github.com/xukgo/gfs/model"
	"net/http"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
)

func (this *Server) RepairStatByDate(date string) model.StatDateFileInfo {
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			log.Error("RepairStatByDate")
			log.Error(re)
			log.Error(string(buffer))
		}
	}()
	var (
		err       error
		keyPrefix string
		fileInfo  model.FileInfo
		fileCount int64
		fileSize  int64
		stat      model.StatDateFileInfo
	)
	keyPrefix = "%s_%s_"
	keyPrefix = fmt.Sprintf(keyPrefix, date, constDefine.CONST_FILE_Md5_FILE_NAME)
	iter := Singleton.logDB.NewIterator(util.BytesPrefix([]byte(keyPrefix)), nil)
	defer iter.Release()
	for iter.Next() {
		if err = json.Unmarshal(iter.Value(), &fileInfo); err != nil {
			continue
		}
		fileCount = fileCount + 1
		fileSize = fileSize + fileInfo.Size
	}
	this.statMap.Put(date+"_"+constDefine.CONST_STAT_FILE_COUNT_KEY, fileCount)
	this.statMap.Put(date+"_"+constDefine.CONST_STAT_FILE_TOTAL_SIZE_KEY, fileSize)
	this.SaveStat()
	stat.Date = date
	stat.FileCount = fileCount
	stat.TotalSize = fileSize
	return stat
}

func (this *Server) FormatStatInfo() {
	var (
		data  []byte
		err   error
		count int64
		stat  map[string]interface{}
	)
	if this.util.FileExists(this.confRepo.GetStatFileName()) {
		if data, err = this.util.ReadBinFile(this.confRepo.GetStatFileName()); err != nil {
			log.Error(err)
		} else {
			if err = json.Unmarshal(data, &stat); err != nil {
				log.Error(err)
			} else {
				for k, v := range stat {
					switch v.(type) {
					case float64:
						vv := strings.Split(fmt.Sprintf("%f", v), ".")[0]
						if count, err = strconv.ParseInt(vv, 10, 64); err != nil {
							log.Error(err)
						} else {
							this.statMap.Put(k, count)
						}
					default:
						this.statMap.Put(k, v)
					}
				}
			}
		}
	} else {
		this.RepairStatByDate(this.util.GetToDay())
	}
}

func (this *Server) RepairStatWeb(w http.ResponseWriter, r *http.Request) {
	var (
		result model.JsonResult
		date   string
		inner  string
	)
	clientIP := this.util.GetClientIp(r)
	if !this.IsPeer(clientIP) {
		result.Message = this.GetClusterNotPermitMessage(clientIP)
		w.Write([]byte(this.util.JsonEncodePretty(result)))
		return
	}
	date = r.FormValue("date")
	inner = r.FormValue("inner")
	if ok, err := regexp.MatchString("\\d{8}", date); err != nil || !ok {
		result.Message = "invalid date"
		w.Write([]byte(this.util.JsonEncodePretty(result)))
		return
	}
	if date == "" || len(date) != 8 {
		date = this.util.GetToDay()
	}
	if inner != "1" {
		for _, peer := range this.confRepo.GetPeers() {
			req := httplib.Post(peer + this.getRequestURI("repair_stat"))
			req.Param("inner", "1")
			req.Param("date", date)
			if _, err := req.String(); err != nil {
				log.Error(err)
			}
		}
	}
	result.Data = this.RepairStatByDate(date)
	result.Status = "ok"
	w.Write([]byte(this.util.JsonEncodePretty(result)))
}

func (this *Server) Stat(w http.ResponseWriter, r *http.Request) {
	var (
		result   model.JsonResult
		inner    string
		echart   string
		category []string
		barCount []int64
		barSize  []int64
		dataMap  map[string]interface{}
	)
	clientIP := this.util.GetClientIp(r)
	if !this.IsPeer(clientIP) {
		result.Message = this.GetClusterNotPermitMessage(clientIP)
		w.Write([]byte(this.util.JsonEncodePretty(result)))
		return
	}
	r.ParseForm()
	inner = r.FormValue("inner")
	echart = r.FormValue("echart")
	data := this.GetStat()
	result.Status = "ok"
	result.Data = data
	if echart == "1" {
		dataMap = make(map[string]interface{}, 3)
		for _, v := range data {
			barCount = append(barCount, v.FileCount)
			barSize = append(barSize, v.TotalSize)
			category = append(category, v.Date)
		}
		dataMap["category"] = category
		dataMap["barCount"] = barCount
		dataMap["barSize"] = barSize
		result.Data = dataMap
	}
	if inner == "1" {
		w.Write([]byte(this.util.JsonEncodePretty(data)))
	} else {
		w.Write([]byte(this.util.JsonEncodePretty(result)))
	}
}
func (this *Server) GetStat() []model.StatDateFileInfo {
	var (
		min   int64
		max   int64
		err   error
		i     int64
		rows  []model.StatDateFileInfo
		total model.StatDateFileInfo
	)
	min = 20190101
	max = 20190101
	for k := range this.statMap.Get() {
		ks := strings.Split(k, "_")
		if len(ks) == 2 {
			if i, err = strconv.ParseInt(ks[0], 10, 64); err != nil {
				continue
			}
			if i >= max {
				max = i
			}
			if i < min {
				min = i
			}
		}
	}
	for i := min; i <= max; i++ {
		s := fmt.Sprintf("%d", i)
		if v, ok := this.statMap.GetValue(s + "_" + constDefine.CONST_STAT_FILE_TOTAL_SIZE_KEY); ok {
			var info model.StatDateFileInfo
			info.Date = s
			switch v.(type) {
			case int64:
				info.TotalSize = v.(int64)
				total.TotalSize = total.TotalSize + v.(int64)
			}
			if v, ok := this.statMap.GetValue(s + "_" + constDefine.CONST_STAT_FILE_COUNT_KEY); ok {
				switch v.(type) {
				case int64:
					info.FileCount = v.(int64)
					total.FileCount = total.FileCount + v.(int64)
				}
			}
			rows = append(rows, info)
		}
	}
	total.Date = "all"
	rows = append(rows, total)
	return rows
}

func (this *Server) Status(w http.ResponseWriter, r *http.Request) {
	var (
		status   model.JsonResult
		sts      map[string]interface{}
		today    string
		sumset   mapset.Set
		ok       bool
		v        interface{}
		err      error
		appDir   string
		diskInfo *disk.UsageStat
		memInfo  *mem.VirtualMemoryStat
	)
	memStat := new(runtime.MemStats)
	runtime.ReadMemStats(memStat)
	today = this.util.GetToDay()
	sts = make(map[string]interface{})
	sts["Fs.QueueFromPeers"] = len(this.queueFromPeers)
	sts["Fs.QueueToPeers"] = len(this.queueToPeers)
	sts["Fs.QueueFileLog"] = len(this.queueFileLog)
	for _, k := range []string{constDefine.CONST_FILE_Md5_FILE_NAME, constDefine.CONST_Md5_ERROR_FILE_NAME, constDefine.CONST_Md5_QUEUE_FILE_NAME} {
		k2 := fmt.Sprintf("%s_%s", today, k)
		if v, ok = this.sumMap.GetValue(k2); ok {
			sumset = v.(mapset.Set)
			if k == constDefine.CONST_Md5_QUEUE_FILE_NAME {
				sts["Fs.QueueSetSize"] = sumset.Cardinality()
			}
			if k == constDefine.CONST_Md5_ERROR_FILE_NAME {
				sts["Fs.ErrorSetSize"] = sumset.Cardinality()
			}
			if k == constDefine.CONST_FILE_Md5_FILE_NAME {
				sts["Fs.FileSetSize"] = sumset.Cardinality()
			}
		}
	}
	sts["Fs.AutoRepair"] = this.confRepo.GetAutoRepair()
	sts["Fs.QueueUpload"] = len(this.queueUpload)
	sts["Fs.RefreshInterval"] = this.confRepo.GetRefreshInterval()
	sts["Fs.Peers"] = this.confRepo.GetPeers()
	sts["Fs.Local"] = this.host
	sts["Fs.FileStats"] = this.GetStat()
	sts["Fs.ShowDir"] = this.confRepo.GetShowDir()
	sts["Sys.NumGoroutine"] = runtime.NumGoroutine()
	sts["Sys.NumCpu"] = runtime.NumCPU()
	sts["Sys.Alloc"] = memStat.Alloc
	sts["Sys.TotalAlloc"] = memStat.TotalAlloc
	sts["Sys.HeapAlloc"] = memStat.HeapAlloc
	sts["Sys.Frees"] = memStat.Frees
	sts["Sys.HeapObjects"] = memStat.HeapObjects
	sts["Sys.NumGC"] = memStat.NumGC
	sts["Sys.GCCPUFraction"] = memStat.GCCPUFraction
	sts["Sys.GCSys"] = memStat.GCSys
	//sts["Sys.MemInfo"] = memStat
	appDir, err = filepath.Abs(".")
	if err != nil {
		log.Error(err)
	}
	diskInfo, err = disk.Usage(appDir)
	if err != nil {
		log.Error(err)
	}
	sts["Sys.DiskInfo"] = diskInfo
	memInfo, err = mem.VirtualMemory()
	if err != nil {
		log.Error(err)
	}
	sts["Sys.MemInfo"] = memInfo
	status.Status = "ok"
	status.Data = sts
	w.Write([]byte(this.util.JsonEncodePretty(status)))
}
