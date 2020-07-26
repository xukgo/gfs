package core

import (
	"bufio"
	"fmt"
	log "github.com/sjqzhang/seelog"
	"github.com/xukgo/gfs/model"
	"net/http"
	"os"
	"strings"
)

// Notice: performance is poor,just for low capacity,but low memory , if you want to high performance,use searchMap for search,but memory ....
func (this *Server) Search(w http.ResponseWriter, r *http.Request) {
	var (
		result    model.JsonResult
		err       error
		kw        string
		count     int
		fileInfos []model.FileInfo
		md5s      []string
	)
	kw = r.FormValue("kw")
	clientIP := this.util.GetClientIp(r)
	if !this.IsPeer(clientIP) {
		result.Message = this.GetClusterNotPermitMessage(clientIP)
		w.Write([]byte(this.util.JsonEncodePretty(result)))
		return
	}
	iter := this.ldb.NewIterator(nil, nil)
	for iter.Next() {
		var fileInfo model.FileInfo
		value := iter.Value()
		if err = json.Unmarshal(value, &fileInfo); err != nil {
			log.Error(err)
			continue
		}
		if strings.Contains(fileInfo.Name, kw) && !this.util.Contains(fileInfo.Md5, md5s) {
			count = count + 1
			fileInfos = append(fileInfos, fileInfo)
			md5s = append(md5s, fileInfo.Md5)
		}
		if count >= 100 {
			break
		}
	}
	iter.Release()
	err = iter.Error()
	if err != nil {
		log.Error()
	}
	//fileInfos=this.SearchDict(kw) // serch file from map for huge capacity
	result.Status = "ok"
	result.Data = fileInfos
	w.Write([]byte(this.util.JsonEncodePretty(result)))
}

func (this *Server) SearchDict(kw string) []model.FileInfo {
	var (
		fileInfos []model.FileInfo
		fileInfo  *model.FileInfo
	)
	for dict := range this.searchMap.Iter() {
		if strings.Contains(dict.Val.(string), kw) {
			if fileInfo, _ = this.GetFileInfoFromLevelDB(dict.Key); fileInfo != nil {
				fileInfos = append(fileInfos, *fileInfo)
			}
		}
	}
	return fileInfos
}

func (this *Server) LoadSearchDict() {
	go func() {
		log.Info("Load search dict ....")
		f, err := os.Open(this.confRepo.GetSearchFileName())
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
	this.lockMap.LockKey(this.confRepo.GetSearchFileName())
	defer this.lockMap.UnLockKey(this.confRepo.GetSearchFileName())
	searchDict = this.searchMap.Get()
	fp, err = os.OpenFile(this.confRepo.GetSearchFileName(), os.O_RDWR, 0755)
	if err != nil {
		log.Error(err)
		return
	}
	defer fp.Close()
	for k, v = range searchDict {
		fp.WriteString(fmt.Sprintf("%s\t%s", k, v.(string)))
	}
}
