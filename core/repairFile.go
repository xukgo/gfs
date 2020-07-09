package core

import (
	"fmt"
	log "github.com/sjqzhang/seelog"
	"github.com/xukgo/gfs/constDefine"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
)

func (this *Server) RepairFileInfoFromFile() {
	var (
		pathPrefix string
		err        error
		fi         os.FileInfo
	)
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			log.Error("RepairFileInfoFromFile")
			log.Error(re)
			log.Error(string(buffer))
		}
	}()
	if this.lockMap.IsLock("RepairFileInfoFromFile") {
		log.Warn("Lock RepairFileInfoFromFile")
		return
	}
	this.lockMap.LockKey("RepairFileInfoFromFile")
	defer this.lockMap.UnLockKey("RepairFileInfoFromFile")
	handlefunc := func(file_path string, f os.FileInfo, err error) error {
		var (
			files    []os.FileInfo
			fi       os.FileInfo
			fileInfo FileInfo
			sum      string
			pathMd5  string
		)
		if f.IsDir() {
			files, err = ioutil.ReadDir(file_path)

			if err != nil {
				return err
			}
			for _, fi = range files {
				if fi.IsDir() || fi.Size() == 0 {
					continue
				}
				file_path = strings.Replace(file_path, "\\", "/", -1)
				if DOCKER_DIR != "" {
					file_path = strings.Replace(file_path, DOCKER_DIR, "", 1)
				}
				if pathPrefix != "" {
					file_path = strings.Replace(file_path, pathPrefix, constDefine.STORE_DIR_NAME, 1)
				}
				if strings.HasPrefix(file_path, constDefine.STORE_DIR_NAME+"/"+LARGE_DIR_NAME) {
					log.Info(fmt.Sprintf("ignore small file file %s", file_path+"/"+fi.Name()))
					continue
				}
				pathMd5 = this.util.MD5(file_path + "/" + fi.Name())
				//if finfo, _ := this.GetFileInfoFromLevelDB(pathMd5); finfo != nil && finfo.Md5 != "" {
				//	log.Info(fmt.Sprintf("exist ignore file %s", file_path+"/"+fi.Name()))
				//	continue
				//}
				//sum, err = this.util.GetFileSumByName(file_path+"/"+fi.Name(), Config().FileSumArithmetic)
				sum = pathMd5
				if err != nil {
					log.Error(err)
					continue
				}
				fileInfo = FileInfo{
					Size:      fi.Size(),
					Name:      fi.Name(),
					Path:      file_path,
					Md5:       sum,
					TimeStamp: fi.ModTime().Unix(),
					Peers:     []string{this.host},
					OffSet:    -2,
				}
				//log.Info(fileInfo)
				log.Info(file_path, "/", fi.Name())
				this.AppendToQueue(&fileInfo)
				//this.postFileToPeer(&fileInfo)
				this.SaveFileInfoToLevelDB(fileInfo.Md5, &fileInfo, this.ldb)
				//this.SaveFileMd5Log(&fileInfo, CONST_FILE_Md5_FILE_NAME)
			}
		}
		return nil
	}
	pathname := STORE_DIR
	pathPrefix, err = os.Readlink(pathname)
	if err == nil {
		//link
		pathname = pathPrefix
		if strings.HasSuffix(pathPrefix, "/") {
			//bugfix fullpath
			pathPrefix = pathPrefix[0 : len(pathPrefix)-1]
		}
	}
	fi, err = os.Stat(pathname)
	if err != nil {
		log.Error(err)
	}
	if fi.IsDir() {
		filepath.Walk(pathname, handlefunc)
	}
	log.Info("RepairFileInfoFromFile is finish.")
}
