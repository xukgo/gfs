package core

import (
	log "github.com/sjqzhang/seelog"
	"github.com/xukgo/gfs/model"
	"net/http"
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
	if !this.IsPeer(r) {
		result.Message = this.GetClusterNotPermitMessage(r)
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
