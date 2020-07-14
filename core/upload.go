package core

import (
	"fmt"
	log "github.com/sjqzhang/seelog"
	"github.com/xukgo/gfs/constDefine"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
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

func (this *Server) SaveUploadFile(file multipart.File, header *multipart.FileHeader, fileInfo *FileInfo, r *http.Request) (*FileInfo, error) {
	var (
		err     error
		outFile *os.File
		folder  string
		fi      os.FileInfo
	)
	defer file.Close()
	_, fileInfo.Name = filepath.Split(header.Filename)
	// bugfix for ie upload file contain fullpath
	if len(Config().Extensions) > 0 && !this.util.Contains(path.Ext(fileInfo.Name), Config().Extensions) {
		return fileInfo, errors.New("(error)file extension mismatch")
	}
	if Config().RenameFile {
		fileInfo.ReName = this.util.MD5(this.util.GetUUID()) + path.Ext(fileInfo.Name)
	}
	folder = time.Now().Format("20060102/15/04")
	if Config().PeerId != "" {
		folder = fmt.Sprintf(folder+"/%s", Config().PeerId)
	}
	if fileInfo.Scene != "" {
		folder = fmt.Sprintf(STORE_DIR+"/%s/%s", fileInfo.Scene, folder)
	} else {
		folder = fmt.Sprintf(STORE_DIR+"/%s", folder)
	}
	if fileInfo.Path != "" {
		if strings.HasPrefix(fileInfo.Path, STORE_DIR) {
			folder = fileInfo.Path
		} else {
			folder = STORE_DIR + "/" + fileInfo.Path
		}
	}
	if !this.util.FileExists(folder) {
		if err = os.MkdirAll(folder, 0775); err != nil {
			log.Error(err)
		}
	}
	outPath := fmt.Sprintf(folder+"/%s", fileInfo.Name)
	if fileInfo.ReName != "" {
		outPath = fmt.Sprintf(folder+"/%s", fileInfo.ReName)
	}
	if this.util.FileExists(outPath) && Config().EnableDistinctFile {
		for i := 0; i < 10000; i++ {
			outPath = fmt.Sprintf(folder+"/%d_%s", i, filepath.Base(header.Filename))
			fileInfo.Name = fmt.Sprintf("%d_%s", i, header.Filename)
			if !this.util.FileExists(outPath) {
				break
			}
		}
	}
	log.Info(fmt.Sprintf("upload: %s", outPath))
	if outFile, err = os.Create(outPath); err != nil {
		return fileInfo, err
	}
	defer outFile.Close()
	if err != nil {
		log.Error(err)
		return fileInfo, errors.New("(error)fail," + err.Error())
	}
	if _, err = io.Copy(outFile, file); err != nil {
		log.Error(err)
		return fileInfo, errors.New("(error)fail," + err.Error())
	}
	if fi, err = outFile.Stat(); err != nil {
		log.Error(err)
		return fileInfo, errors.New("(error)fail," + err.Error())
	} else {
		fileInfo.Size = fi.Size()
	}
	if fi.Size() != header.Size {
		return fileInfo, errors.New("(error)file uncomplete")
	}
	v := "" // this.util.GetFileSum(outFile, Config().FileSumArithmetic)
	if Config().EnableDistinctFile {
		v = this.util.GetFileSum(outFile, Config().FileSumArithmetic)
	} else {
		v = this.util.MD5(this.GetFilePathByInfo(fileInfo, false))
	}
	fileInfo.Md5 = v
	//fileInfo.Path = folder //strings.Replace( folder,DOCKER_DIR,"",1)
	fileInfo.Path = strings.Replace(folder, DOCKER_DIR, "", 1)
	fileInfo.Peers = append(fileInfo.Peers, this.host)
	//fmt.Println("upload",fileInfo)
	return fileInfo, nil
}

func (this *Server) SaveSmallFile(fileInfo *FileInfo) error {
	var (
		err      error
		filename string
		fpath    string
		srcFile  *os.File
		desFile  *os.File
		largeDir string
		destPath string
		reName   string
		fileExt  string
	)
	filename = fileInfo.Name
	fileExt = path.Ext(filename)
	if fileInfo.ReName != "" {
		filename = fileInfo.ReName
	}
	fpath = DOCKER_DIR + fileInfo.Path + "/" + filename
	largeDir = LARGE_DIR + "/" + Config().PeerId
	if !this.util.FileExists(largeDir) {
		os.MkdirAll(largeDir, 0775)
	}
	reName = fmt.Sprintf("%d", this.util.RandInt(100, 300))
	destPath = largeDir + "/" + reName
	this.lockMap.LockKey(destPath)
	defer this.lockMap.UnLockKey(destPath)
	if this.util.FileExists(fpath) {
		srcFile, err = os.OpenFile(fpath, os.O_CREATE|os.O_RDONLY, 06666)
		if err != nil {
			return err
		}
		defer srcFile.Close()
		desFile, err = os.OpenFile(destPath, os.O_CREATE|os.O_RDWR, 06666)
		if err != nil {
			return err
		}
		defer desFile.Close()
		fileInfo.OffSet, err = desFile.Seek(0, 2)
		if _, err = desFile.Write([]byte("1")); err != nil {
			//first byte set 1
			return err
		}
		fileInfo.OffSet, err = desFile.Seek(0, 2)
		if err != nil {
			return err
		}
		fileInfo.OffSet = fileInfo.OffSet - 1 //minus 1 byte
		fileInfo.Size = fileInfo.Size + 1
		fileInfo.ReName = fmt.Sprintf("%s,%d,%d,%s", reName, fileInfo.OffSet, fileInfo.Size, fileExt)
		if _, err = io.Copy(desFile, srcFile); err != nil {
			return err
		}
		srcFile.Close()
		os.Remove(fpath)
		fileInfo.Path = strings.Replace(largeDir, DOCKER_DIR, "", 1)
	}
	return nil
}

func (this *Server) CheckScene(scene string) (bool, error) {
	var (
		scenes []string
	)
	if len(Config().Scenes) == 0 {
		return true, nil
	}
	for _, s := range Config().Scenes {
		scenes = append(scenes, strings.Split(s, ":")[0])
	}
	if !this.util.Contains(scene, scenes) {
		return false, errors.New("not valid scene")
	}
	return true, nil
}
