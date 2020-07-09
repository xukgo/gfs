package core

import (
	log "github.com/sjqzhang/seelog"
	"github.com/xukgo/gfs/constDefine"
	"mime/multipart"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

func (this *Server) ConsumerUpload() {
	ConsumerFunc := func() {
		for {
			wr := <-this.queueUpload
			this.upload(*wr.w, wr.r)
			this.rtMap.AddCountInt64(CONST_UPLOAD_COUNTER_KEY, wr.r.ContentLength)
			if v, ok := this.rtMap.GetValue(CONST_UPLOAD_COUNTER_KEY); ok {
				if v.(int64) > 1*1024*1024*1024 {
					var _v int64
					this.rtMap.Put(CONST_UPLOAD_COUNTER_KEY, _v)
					debug.FreeOSMemory()
				}
			}
			wr.done <- true
		}
	}
	for i := 0; i < Config().UploadWorker; i++ {
		go ConsumerFunc()
	}
}

func (this *Server) upload(w http.ResponseWriter, r *http.Request) {
	var (
		err error
		//		pathname     string
		md5sum       string
		fileName     string
		fileInfo     FileInfo
		uploadFile   multipart.File
		uploadHeader *multipart.FileHeader
		scene        string
		output       string
		fileResult   FileResult
		result       JsonResult
		data         []byte
		msg          string
	)
	output = r.FormValue("output")
	if Config().EnableCrossOrigin {
		this.CrossOrigin(w, r)
		if r.Method == http.MethodOptions {
			return
		}
	}
	result.Status = "fail"
	if Config().AuthUrl != "" {
		if !this.CheckAuth(w, r) {
			msg = "auth fail"
			log.Warn(msg, r.Form)
			this.NotPermit(w, r)
			result.Message = msg
			w.Write([]byte(this.util.JsonEncodePretty(result)))
			return
		}
	}
	if r.Method == http.MethodPost {
		md5sum = r.FormValue("md5")
		fileName = r.FormValue("filename")
		output = r.FormValue("output")
		if Config().EnableCustomPath {
			fileInfo.Path = r.FormValue("path")
			fileInfo.Path = strings.Trim(fileInfo.Path, "/")
		}
		scene = r.FormValue("scene")
		if scene == "" {
			//Just for Compatibility
			scene = r.FormValue("scenes")
		}
		fileInfo.Md5 = md5sum
		fileInfo.ReName = fileName
		fileInfo.OffSet = -1
		if uploadFile, uploadHeader, err = r.FormFile("file"); err != nil {
			log.Error(err)
			result.Message = err.Error()
			w.Write([]byte(this.util.JsonEncodePretty(result)))
			return
		}
		fileInfo.Peers = []string{}
		fileInfo.TimeStamp = time.Now().Unix()
		if scene == "" {
			scene = Config().DefaultScene
		}
		if output == "" {
			output = "text"
		}
		if !this.util.Contains(output, []string{"json", "text", "json2"}) {
			msg = "output just support json or text or json2"
			result.Message = msg
			log.Warn(msg)
			w.Write([]byte(this.util.JsonEncodePretty(result)))
			return
		}
		fileInfo.Scene = scene
		if _, err = this.CheckScene(scene); err != nil {
			result.Message = err.Error()
			w.Write([]byte(this.util.JsonEncodePretty(result)))
			log.Error(err)
			return
		}
		if err != nil {
			log.Error(err)
			http.Redirect(w, r, "/", http.StatusMovedPermanently)
			return
		}
		if _, err = this.SaveUploadFile(uploadFile, uploadHeader, &fileInfo, r); err != nil {
			result.Message = err.Error()
			log.Error(err)
			w.Write([]byte(this.util.JsonEncodePretty(result)))
			return
		}
		if Config().EnableDistinctFile {
			if v, _ := this.GetFileInfoFromLevelDB(fileInfo.Md5); v != nil && v.Md5 != "" {
				fileResult = this.BuildFileResult(v, r)
				if Config().RenameFile {
					os.Remove(DOCKER_DIR + fileInfo.Path + "/" + fileInfo.ReName)
				} else {
					os.Remove(DOCKER_DIR + fileInfo.Path + "/" + fileInfo.Name)
				}
				if output == "json" || output == "json2" {
					if output == "json2" {
						result.Data = fileResult
						result.Status = "ok"
						w.Write([]byte(this.util.JsonEncodePretty(result)))
						return
					}
					w.Write([]byte(this.util.JsonEncodePretty(fileResult)))
				} else {
					w.Write([]byte(fileResult.Url))
				}
				return
			}
		}
		if fileInfo.Md5 == "" {
			msg = " fileInfo.Md5 is null"
			log.Warn(msg)
			result.Message = msg
			w.Write([]byte(this.util.JsonEncodePretty(result)))
			return
		}
		if md5sum != "" && fileInfo.Md5 != md5sum {
			msg = " fileInfo.Md5 and md5sum !="
			log.Warn(msg)
			result.Message = msg
			w.Write([]byte(this.util.JsonEncodePretty(result)))
			return
		}
		if !Config().EnableDistinctFile {
			// bugfix filecount stat
			fileInfo.Md5 = this.util.MD5(this.GetFilePathByInfo(&fileInfo, false))
		}
		if Config().EnableMergeSmallFile && fileInfo.Size < constDefine.CONST_SMALL_FILE_SIZE {
			if err = this.SaveSmallFile(&fileInfo); err != nil {
				log.Error(err)
				result.Message = err.Error()
				w.Write([]byte(this.util.JsonEncodePretty(result)))
				return
			}
		}
		this.saveFileMd5Log(&fileInfo, constDefine.CONST_FILE_Md5_FILE_NAME) //maybe slow
		go this.postFileToPeer(&fileInfo)
		if fileInfo.Size <= 0 {
			msg = "file size is zero"
			result.Message = msg
			w.Write([]byte(this.util.JsonEncodePretty(result)))
			log.Error(msg)
			return
		}
		fileResult = this.BuildFileResult(&fileInfo, r)

		if output == "json" || output == "json2" {
			if output == "json2" {
				result.Data = fileResult
				result.Status = "ok"
				w.Write([]byte(this.util.JsonEncodePretty(result)))
				return
			}
			w.Write([]byte(this.util.JsonEncodePretty(fileResult)))
		} else {
			w.Write([]byte(fileResult.Url))
		}
		return
	} else {
		md5sum = r.FormValue("md5")
		output = r.FormValue("output")
		if md5sum == "" {
			msg = "(error) if you want to upload fast md5 is require" +
				",and if you want to upload file,you must use post method  "
			result.Message = msg
			log.Error(msg)
			w.Write([]byte(this.util.JsonEncodePretty(result)))
			return
		}
		if v, _ := this.GetFileInfoFromLevelDB(md5sum); v != nil && v.Md5 != "" {
			fileResult = this.BuildFileResult(v, r)
			result.Data = fileResult
			result.Status = "ok"
		}
		if output == "json" || output == "json2" {
			if data, err = json.Marshal(fileResult); err != nil {
				log.Error(err)
				result.Message = err.Error()
				w.Write([]byte(this.util.JsonEncodePretty(result)))
				return
			}
			if output == "json2" {
				w.Write([]byte(this.util.JsonEncodePretty(result)))
				return
			}
			w.Write(data)
		} else {
			w.Write([]byte(fileResult.Url))
		}
	}
}
