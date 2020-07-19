package core

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/xukgo/gfs/model"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

func (this *Server) BenchMark(w http.ResponseWriter, r *http.Request) {
	t := time.Now()
	batch := new(leveldb.Batch)
	for i := 0; i < 100000000; i++ {
		f := model.FileInfo{}
		f.Peers = []string{"http://192.168.0.1", "http://192.168.2.5"}
		f.Path = "20190201/19/02"
		s := strconv.Itoa(i)
		s = this.util.MD5(s)
		f.Name = s
		f.Md5 = s
		if data, err := json.Marshal(&f); err == nil {
			batch.Put([]byte(s), data)
		}
		if i%10000 == 0 {
			if batch.Len() > 0 {
				Singleton.ldb.Write(batch, nil)
				//				batch = new(leveldb.Batch)
				batch.Reset()
			}
			fmt.Println(i, time.Since(t).Seconds())
		}
		//fmt.Println(Singleton.GetFileInfoFromLevelDB(s))
	}
	this.util.WriteFile("time.txt", time.Since(t).String())
	fmt.Println(time.Since(t).String())
}
func (this *Server) test() {

	testLock := func() {
		wg := sync.WaitGroup{}
		tt := func(i int, wg *sync.WaitGroup) {
			//if Singleton.lockMap.IsLock("xx") {
			//	return
			//}
			//fmt.Println("timeer len",len(Singleton.lockMap.Get()))
			//time.Sleep(time.Nanosecond*10)
			Singleton.lockMap.LockKey("xx")
			defer Singleton.lockMap.UnLockKey("xx")
			//time.Sleep(time.Nanosecond*1)
			//fmt.Println("xx", i)
			wg.Done()
		}
		go func() {
			for {
				time.Sleep(time.Second * 1)
				fmt.Println("timeer len", len(Singleton.lockMap.Get()), Singleton.lockMap.Get())
			}
		}()
		fmt.Println(len(Singleton.lockMap.Get()))
		for i := 0; i < 10000; i++ {
			wg.Add(1)
			go tt(i, &wg)
		}
		fmt.Println(len(Singleton.lockMap.Get()))
		fmt.Println(len(Singleton.lockMap.Get()))
		Singleton.lockMap.LockKey("abc")
		fmt.Println("lock")
		time.Sleep(time.Second * 5)
		Singleton.lockMap.UnLockKey("abc")
		Singleton.lockMap.LockKey("abc")
		Singleton.lockMap.UnLockKey("abc")
	}
	_ = testLock
	testFile := func() {
		var (
			err error
			f   *os.File
		)
		f, err = os.OpenFile("tt", os.O_CREATE|os.O_RDWR, 0777)
		if err != nil {
			fmt.Println(err)
		}
		f.WriteAt([]byte("1"), 100)
		f.Seek(0, 2)
		f.Write([]byte("2"))
		//fmt.Println(f.Seek(0, 2))
		//fmt.Println(f.Seek(3, 2))
		//fmt.Println(f.Seek(3, 0))
		//fmt.Println(f.Seek(3, 1))
		//fmt.Println(f.Seek(3, 0))
		//f.Write([]byte("1"))
	}
	_ = testFile
	//testFile()
	//testLock()
}
