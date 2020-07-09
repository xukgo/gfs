package core

import (
	"errors"
	"fmt"
	mapset "github.com/deckarep/golang-set"
	log "github.com/sjqzhang/seelog"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/xukgo/gfs/constDefine"
	"runtime/debug"
)

func (this *Server) SaveFileInfoToLevelDB(key string, fileInfo *FileInfo, db *leveldb.DB) (*FileInfo, error) {
	var (
		err  error
		data []byte
	)
	if fileInfo == nil || db == nil {
		return nil, errors.New("fileInfo is null or db is null")
	}
	if data, err = json.Marshal(fileInfo); err != nil {
		return fileInfo, err
	}
	if err = db.Put([]byte(key), data, nil); err != nil {
		return fileInfo, err
	}
	if db == this.ldb { //search slow ,write fast, double write logDB
		logDate := this.util.GetDayFromTimeStamp(fileInfo.TimeStamp)
		logKey := fmt.Sprintf("%s_%s_%s", logDate, constDefine.CONST_FILE_Md5_FILE_NAME, fileInfo.Md5)
		this.logDB.Put([]byte(logKey), data, nil)
	}
	return fileInfo, nil
}

func (this *Server) LoadFileInfoByDate(date string, filename string) (mapset.Set, error) {
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			log.Error("LoadFileInfoByDate")
			log.Error(re)
			log.Error(string(buffer))
		}
	}()
	var (
		err       error
		keyPrefix string
		fileInfos mapset.Set
	)
	fileInfos = mapset.NewSet()
	keyPrefix = "%s_%s_"
	keyPrefix = fmt.Sprintf(keyPrefix, date, filename)
	iter := Singleton.logDB.NewIterator(util.BytesPrefix([]byte(keyPrefix)), nil)
	for iter.Next() {
		var fileInfo FileInfo
		if err = json.Unmarshal(iter.Value(), &fileInfo); err != nil {
			continue
		}
		fileInfos.Add(&fileInfo)
	}
	iter.Release()
	return fileInfos, nil
}
