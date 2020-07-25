package core

import (
	log "github.com/sjqzhang/seelog"
	"github.com/xukgo/gfs/model"
	"net/http"
	"os"
	"path"
	"strings"
)

func (this *Server) CheckFileExist(w http.ResponseWriter, r *http.Request) {
	var (
		data     []byte
		err      error
		fileInfo *model.FileInfo
		fpath    string
		fi       os.FileInfo
	)
	r.ParseForm()
	md5sum := ""
	md5sum = r.FormValue("md5")
	fpath = r.FormValue("path")
	if fileInfo, err = this.GetFileInfoFromLevelDB(md5sum); fileInfo != nil {
		if fileInfo.OffSet != -1 {
			if data, err = json.Marshal(fileInfo); err != nil {
				log.Error(err)
			}
			w.Write(data)
			return
		}
		fpath = this.confRepo.GetDockerDir() + fileInfo.Path + "/" + fileInfo.Name
		if fileInfo.ReName != "" {
			fpath = this.confRepo.GetDockerDir() + fileInfo.Path + "/" + fileInfo.ReName
		}
		if this.util.IsExist(fpath) {
			if data, err = json.Marshal(fileInfo); err == nil {
				w.Write(data)
				return
			} else {
				log.Error(err)
			}
		} else {
			if fileInfo.OffSet == -1 {
				this.RemoveKeyFromLevelDB(md5sum, this.ldb) // when file delete,delete from leveldb
			}
		}
	} else {
		if fpath != "" {
			fi, err = os.Stat(fpath)
			if err == nil {
				sum := this.util.MD5(fpath)
				//if this.confRepo.GetEnableDistinctFile() {
				//	sum, err = this.util.GetFileSumByName(fpath, this.confRepo.GetFileSumArithmetic())
				//	if err != nil {
				//		log.Error(err)
				//	}
				//}
				fileInfo = &model.FileInfo{
					Path:      path.Dir(fpath),
					Name:      path.Base(fpath),
					Size:      fi.Size(),
					Md5:       sum,
					Peers:     []string{this.confRepo.GetHost()},
					OffSet:    -1, //very important
					TimeStamp: fi.ModTime().Unix(),
				}
				data, err = json.Marshal(fileInfo)
				w.Write(data)
				return
			}
		}
	}
	data, _ = json.Marshal(model.FileInfo{})
	w.Write(data)
	return
}
func (this *Server) CheckFilesExist(w http.ResponseWriter, r *http.Request) {
	var (
		data      []byte
		err       error
		fileInfo  *model.FileInfo
		fileInfos []*model.FileInfo
		fpath     string
		result    model.JsonResult
	)
	r.ParseForm()
	md5sum := ""
	md5sum = r.FormValue("md5s")
	md5s := strings.Split(md5sum, ",")
	for _, m := range md5s {
		if fileInfo, err = this.GetFileInfoFromLevelDB(m); fileInfo != nil {
			if fileInfo.OffSet != -1 {
				if data, err = json.Marshal(fileInfo); err != nil {
					log.Error(err)
				}
				//w.Write(data)
				//return
				fileInfos = append(fileInfos, fileInfo)
				continue
			}
			fpath = this.confRepo.GetDockerDir() + fileInfo.Path + "/" + fileInfo.Name
			if fileInfo.ReName != "" {
				fpath = this.confRepo.GetDockerDir() + fileInfo.Path + "/" + fileInfo.ReName
			}
			if this.util.IsExist(fpath) {
				if data, err = json.Marshal(fileInfo); err == nil {
					fileInfos = append(fileInfos, fileInfo)
					//w.Write(data)
					//return
					continue
				} else {
					log.Error(err)
				}
			} else {
				if fileInfo.OffSet == -1 {
					this.RemoveKeyFromLevelDB(md5sum, this.ldb) // when file delete,delete from leveldb
				}
			}
		}
	}
	result.Data = fileInfos
	data, _ = json.Marshal(result)
	w.Write(data)
	return
}
