package core

import (
	"fmt"
	"github.com/astaxie/beego/httplib"
	log "github.com/sjqzhang/seelog"
	"github.com/xukgo/gfs/constDefine"
	"github.com/xukgo/gfs/model"
	"os"
	"strings"
	"time"
)

func (this *Server) DownloadFromPeer(peer string, fileInfo *model.FileInfo) {
	var (
		err         error
		filename    string
		fpath       string
		fpathTmp    string
		fi          os.FileInfo
		sum         string
		data        []byte
		downloadUrl string
	)
	if this.confRepo.GetRetryCount() > 0 && fileInfo.Retry >= this.confRepo.GetRetryCount() {
		log.Error("DownloadFromPeer Error ", fileInfo)
		return
	} else {
		fileInfo.Retry = fileInfo.Retry + 1
	}
	filename = fileInfo.Name
	if fileInfo.ReName != "" {
		filename = fileInfo.ReName
	}
	//如果已经存在且不允许重复文件则返回
	if fileInfo.OffSet != -2 && this.confRepo.GetEnableDistinctFile() && this.CheckFileExistByInfo(fileInfo.Md5, fileInfo) {
		// ignore migrate file
		log.Info(fmt.Sprintf("DownloadFromPeer file Exist, path:%s", fileInfo.Path+"/"+fileInfo.Name))
		return
	}
	if (!this.confRepo.GetEnableDistinctFile() || fileInfo.OffSet == -2) && this.util.FileExists(this.GetFilePathByInfo(fileInfo, true)) {
		// ignore migrate file
		if fi, err = os.Stat(this.GetFilePathByInfo(fileInfo, true)); err == nil {
			if fi.ModTime().Unix() > fileInfo.TimeStamp {
				log.Info(fmt.Sprintf("ignore file sync path:%s", this.GetFilePathByInfo(fileInfo, false)))
				fileInfo.TimeStamp = fi.ModTime().Unix()
				this.postFileToPeer(fileInfo) // keep newer
				return
			}
			os.Remove(this.GetFilePathByInfo(fileInfo, true))
		}
	}
	if _, err = os.Stat(fileInfo.Path); err != nil {
		os.MkdirAll(DOCKER_DIR+fileInfo.Path, 0775)
	}
	//fmt.Println("downloadFromPeer",fileInfo)
	p := strings.Replace(fileInfo.Path, constDefine.STORE_DIR_NAME+"/", "", 1)
	//filename=this.util.UrlEncode(filename)
	downloadUrl = peer + "/" + this.confRepo.GetGroup() + "/" + p + "/" + filename
	log.Info("DownloadFromPeer: ", downloadUrl)
	fpath = DOCKER_DIR + fileInfo.Path + "/" + filename
	fpathTmp = DOCKER_DIR + fileInfo.Path + "/" + fmt.Sprintf("%s_%s", "tmp_", filename)
	timeout := fileInfo.Size/1024/1024/1 + 20
	if this.confRepo.GetSyncTimeout() > 0 {
		timeout = this.confRepo.GetSyncTimeout()
	}
	this.lockMap.LockKey(fpath)
	defer this.lockMap.UnLockKey(fpath)
	download_key := fmt.Sprintf("downloading_%d_%s", time.Now().Unix(), fpath)
	this.ldb.Put([]byte(download_key), []byte(""), nil)
	defer func() {
		this.ldb.Delete([]byte(download_key), nil)
	}()
	if fileInfo.OffSet == -2 {
		//migrate file
		if fi, err = os.Stat(fpath); err == nil && fi.Size() == fileInfo.Size {
			//prevent double download
			this.SaveFileInfoToLevelDB(fileInfo.Md5, fileInfo, this.ldb)
			//log.Info(fmt.Sprintf("file '%s' has download", fpath))
			return
		}
		//下载文件，先下载到临时文件，然后重命名
		req := httplib.Get(downloadUrl)
		req.SetTimeout(time.Second*30, time.Second*time.Duration(timeout))
		if err = req.ToFile(fpathTmp); err != nil {
			this.AppendToDownloadQueue(fileInfo) //retry
			os.Remove(fpathTmp)
			log.Error(err, fpathTmp)
			return
		}
		if os.Rename(fpathTmp, fpath) == nil {
			//this.SaveFileMd5Log(fileInfo, CONST_FILE_Md5_FILE_NAME)
			this.SaveFileInfoToLevelDB(fileInfo.Md5, fileInfo, this.ldb)
		}
		return
	}

	req := httplib.Get(downloadUrl)
	req.SetTimeout(time.Second*30, time.Second*time.Duration(timeout))
	if fileInfo.OffSet >= 0 {
		//small file download
		data, err = req.Bytes()
		if err != nil {
			this.AppendToDownloadQueue(fileInfo) //retry
			log.Error(err)
			return
		}
		data2 := make([]byte, len(data)+1)
		data2[0] = '1'
		for i, v := range data {
			data2[i+1] = v
		}
		data = data2
		if int64(len(data)) != fileInfo.Size {
			log.Warn("file size is error")
			return
		}
		fpath = strings.Split(fpath, ",")[0]
		err = this.util.WriteFileByOffSet(fpath, fileInfo.OffSet, data)
		if err != nil {
			log.Warn(err)
			return
		}
		this.SaveFileMd5Log(fileInfo, constDefine.CONST_FILE_Md5_FILE_NAME)
		return
	}
	if err = req.ToFile(fpathTmp); err != nil {
		this.AppendToDownloadQueue(fileInfo) //retry
		os.Remove(fpathTmp)
		log.Error(err)
		return
	}
	if fi, err = os.Stat(fpathTmp); err != nil {
		os.Remove(fpathTmp)
		return
	}
	_ = sum
	//if this.confRepo.GetEnableDistinctFile() {
	//	//DistinctFile
	//	if sum, err = this.util.GetFileSumByName(fpathTmp, this.confRepo.GetFileSumArithmetic()); err != nil {
	//		log.Error(err)
	//		return
	//	}
	//} else {
	//	//DistinctFile By path
	//	sum = this.util.MD5(this.GetFilePathByInfo(fileInfo, false))
	//}
	if fi.Size() != fileInfo.Size { //  maybe has bug remove || sum != fileInfo.Md5
		log.Error("file sum check error")
		os.Remove(fpathTmp)
		return
	}
	if os.Rename(fpathTmp, fpath) == nil {
		this.SaveFileMd5Log(fileInfo, constDefine.CONST_FILE_Md5_FILE_NAME)
	}
}

func (this *Server) CheckFileExistByInfo(md5s string, fileInfo *model.FileInfo) bool {
	var (
		err      error
		fullpath string
		fi       os.FileInfo
		info     *model.FileInfo
	)
	if fileInfo == nil {
		return false
	}
	if fileInfo.OffSet >= 0 {
		//small file
		if info, err = this.GetFileInfoFromLevelDB(fileInfo.Md5); err == nil && info.Md5 == fileInfo.Md5 {
			return true
		} else {
			return false
		}
	}
	fullpath = this.GetFilePathByInfo(fileInfo, true)
	if fi, err = os.Stat(fullpath); err != nil {
		return false
	}
	if fi.Size() == fileInfo.Size {
		return true
	} else {
		return false
	}
}
