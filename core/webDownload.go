package core

import (
	"bytes"
	"errors"
	"github.com/astaxie/beego/httplib"
	"github.com/nfnt/resize"
	log "github.com/sjqzhang/seelog"
	"github.com/xukgo/gfs/constDefine"
	"github.com/xukgo/gfs/model"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func (this *Server) Download(w http.ResponseWriter, r *http.Request) {
	var (
		err       error
		ok        bool
		fullpath  string
		smallPath string
		fi        os.FileInfo
	)
	// redirect to upload
	if r.RequestURI == "/" || r.RequestURI == "" ||
		r.RequestURI == "/"+this.confRepo.GetGroup() ||
		r.RequestURI == "/"+this.confRepo.GetGroup()+"/" {
		this.Index(w, r)
		return
	}
	if ok, err = this.CheckDownloadAuth(w, r); !ok {
		log.Error(err)
		this.NotPermit(w, r)
		return
	}

	if this.confRepo.GetEnableCrossOrigin() {
		this.CrossOrigin(w, r)
	}
	fullpath, smallPath = this.GetFilePathFromRequest(w, r)
	if smallPath == "" {
		if fi, err = os.Stat(fullpath); err != nil {
			this.DownloadNotFound(w, r)
			return
		}
		if !this.confRepo.GetShowDir() && fi.IsDir() {
			w.Write([]byte("list dir deny"))
			return
		}
		//staticFileServerHandler.ServeHTTP(w, r)
		this.DownloadNormalFileByURI(w, r)
		return
	}
	if smallPath != "" {
		if ok, err = this.DownloadSmallFileByURI(w, r); !ok {
			this.DownloadNotFound(w, r)
			return
		}
		return
	}
}

func (this *Server) DownloadSmallFileByURI(w http.ResponseWriter, r *http.Request) (bool, error) {
	var (
		err        error
		data       []byte
		isDownload bool
		imgWidth   int
		imgHeight  int
		width      string
		height     string
		notFound   bool
	)
	r.ParseForm()
	isDownload = true
	if r.FormValue("download") == "" {
		isDownload = this.confRepo.GetDefaultDownload()
	}
	if r.FormValue("download") == "0" {
		isDownload = false
	}
	width = r.FormValue("width")
	height = r.FormValue("height")
	if width != "" {
		imgWidth, err = strconv.Atoi(width)
		if err != nil {
			log.Error(err)
		}
	}
	if height != "" {
		imgHeight, err = strconv.Atoi(height)
		if err != nil {
			log.Error(err)
		}
	}
	data, notFound, err = this.GetSmallFileByURI(w, r)
	_ = notFound
	if data != nil && string(data[0]) == "1" {
		if isDownload {
			this.SetDownloadHeader(w, r)
		}
		if imgWidth != 0 || imgHeight != 0 {
			this.ResizeImageByBytes(w, data[1:], uint(imgWidth), uint(imgHeight))
			return true, nil
		}
		w.Write(data[1:])
		return true, nil
	}
	return false, errors.New("not found")
}
func (this *Server) DownloadNormalFileByURI(w http.ResponseWriter, r *http.Request) (bool, error) {
	var (
		err        error
		isDownload bool
		imgWidth   int
		imgHeight  int
		width      string
		height     string
	)
	r.ParseForm()
	isDownload = true
	if r.FormValue("download") == "" {
		isDownload = this.confRepo.GetDefaultDownload()
	}
	if r.FormValue("download") == "0" {
		isDownload = false
	}
	width = r.FormValue("width")
	height = r.FormValue("height")
	if width != "" {
		imgWidth, err = strconv.Atoi(width)
		if err != nil {
			log.Error(err)
		}
	}
	if height != "" {
		imgHeight, err = strconv.Atoi(height)
		if err != nil {
			log.Error(err)
		}
	}
	if isDownload {
		this.SetDownloadHeader(w, r)
	}
	fullpath, _ := this.GetFilePathFromRequest(w, r)
	if imgWidth != 0 || imgHeight != 0 {
		this.ResizeImage(w, fullpath, uint(imgWidth), uint(imgHeight))
		return true, nil
	}
	staticFileServerHandler.ServeHTTP(w, r)
	return true, nil
}
func (this *Server) DownloadNotFound(w http.ResponseWriter, r *http.Request) {
	var (
		err        error
		fullpath   string
		smallPath  string
		isDownload bool
		pathMd5    string
		peer       string
		fileInfo   *model.FileInfo
	)
	fullpath, smallPath = this.GetFilePathFromRequest(w, r)
	isDownload = true
	if r.FormValue("download") == "" {
		isDownload = this.confRepo.GetDefaultDownload()
	}
	if r.FormValue("download") == "0" {
		isDownload = false
	}
	if smallPath != "" {
		pathMd5 = this.util.MD5(smallPath)
	} else {
		pathMd5 = this.util.MD5(fullpath)
	}
	for _, peer = range this.confRepo.GetPeers() {
		if fileInfo, err = this.checkPeerFileExist(peer, pathMd5, fullpath); err != nil {
			log.Error(err)
			continue
		}
		if fileInfo.Md5 != "" {
			go this.DownloadFromPeer(peer, fileInfo)
			//http.Redirect(w, r, peer+r.RequestURI, 302)
			if isDownload {
				this.SetDownloadHeader(w, r)
			}
			this.DownloadFileToResponse(peer+r.RequestURI, w, r)
			return
		}
	}
	w.WriteHeader(404)
	return
}
func (this *Server) DownloadFileToResponse(url string, w http.ResponseWriter, r *http.Request) {
	var (
		err  error
		req  *httplib.BeegoHTTPRequest
		resp *http.Response
	)
	req = httplib.Get(url)
	req.SetTimeout(time.Second*20, time.Second*600)
	resp, err = req.DoRequest()
	if err != nil {
		log.Error(err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Error(err)
	}
}

func (this *Server) GetSmallFileByURI(w http.ResponseWriter, r *http.Request) ([]byte, bool, error) {
	var (
		err      error
		data     []byte
		offset   int64
		length   int
		fullpath string
		info     os.FileInfo
	)
	fullpath, _ = this.GetFilePathFromRequest(w, r)
	if _, offset, length, err = this.ParseSmallFile(r.RequestURI); err != nil {
		return nil, false, err
	}
	if info, err = os.Stat(fullpath); err != nil {
		return nil, false, err
	}
	if info.Size() < offset+int64(length) {
		return nil, true, errors.New("noFound")
	} else {
		data, err = this.util.ReadFileByOffSet(fullpath, offset, length)
		if err != nil {
			return nil, false, err
		}
		return data, false, err
	}
}

func (this *Server) CheckDownloadAuth(w http.ResponseWriter, r *http.Request) (bool, error) {
	var (
		err          error
		maxTimestamp int64
		minTimestamp int64
		ts           int64
		token        string
		timestamp    string
		fullpath     string
		smallPath    string
		pathMd5      string
		fileInfo     *model.FileInfo
	)
	CheckToken := func(token string, md5sum string, timestamp string) bool {
		if this.util.MD5(md5sum+timestamp) != token {
			return false
		}
		return true
	}
	clientIP := this.util.GetClientIp(r)
	if this.confRepo.GetEnableDownloadAuth() && this.confRepo.GetAuthUrl() != "" && !this.IsPeer(clientIP) && !this.CheckAuth(w, r) {
		return false, errors.New("auth fail")
	}
	if this.confRepo.GetDownloadUseToken() && !this.IsPeer(clientIP) {
		token = r.FormValue("token")
		timestamp = r.FormValue("timestamp")
		if token == "" || timestamp == "" {
			return false, errors.New("unvalid request")
		}
		maxTimestamp = time.Now().Add(time.Second *
			time.Duration(this.confRepo.GetDownloadTokenExpire())).Unix()
		minTimestamp = time.Now().Add(-time.Second *
			time.Duration(this.confRepo.GetDownloadTokenExpire())).Unix()
		if ts, err = strconv.ParseInt(timestamp, 10, 64); err != nil {
			return false, errors.New("unvalid timestamp")
		}
		if ts > maxTimestamp || ts < minTimestamp {
			return false, errors.New("timestamp expire")
		}
		fullpath, smallPath = this.GetFilePathFromRequest(w, r)
		if smallPath != "" {
			pathMd5 = this.util.MD5(smallPath)
		} else {
			pathMd5 = this.util.MD5(fullpath)
		}
		if fileInfo, err = this.GetFileInfoFromLevelDB(pathMd5); err != nil {
			// TODO
		} else {
			ok := CheckToken(token, fileInfo.Md5, timestamp)
			if !ok {
				return ok, errors.New("unvalid token")
			}
			return ok, nil
		}
	}
	return true, nil
}

func (this *Server) GetFilePathFromRequest(w http.ResponseWriter, r *http.Request) (string, string) {
	var (
		err       error
		fullpath  string
		smallPath string
		prefix    string
	)
	fullpath = r.RequestURI[1:]
	if strings.HasPrefix(r.RequestURI, "/"+this.confRepo.GetGroup()+"/") {
		fullpath = r.RequestURI[len(this.confRepo.GetGroup())+2 : len(r.RequestURI)]
	}
	fullpath = strings.Split(fullpath, "?")[0] // just path
	fullpath = this.confRepo.GetDockerDir() + constDefine.STORE_DIR_NAME + "/" + fullpath
	if this.confRepo.GetSupportGroupManage() {
		prefix = "/" + this.confRepo.GetGroup() + "/" + this.confRepo.GetLargeDirName() + "/"
	} else {
		prefix = "/" + this.confRepo.GetLargeDirName() + "/"
	}
	if strings.HasPrefix(r.RequestURI, prefix) {
		smallPath = fullpath //notice order
		fullpath = strings.Split(fullpath, ",")[0]
	}
	if fullpath, err = url.PathUnescape(fullpath); err != nil {
		log.Error(err)
	}
	return fullpath, smallPath
}

func (this *Server) ResizeImageByBytes(w http.ResponseWriter, data []byte, width, height uint) {
	var (
		img     image.Image
		err     error
		imgType string
	)
	reader := bytes.NewReader(data)
	img, imgType, err = image.Decode(reader)
	if err != nil {
		log.Error(err)
		return
	}
	img = resize.Resize(width, height, img, resize.Lanczos3)
	if imgType == "jpg" || imgType == "jpeg" {
		jpeg.Encode(w, img, nil)
	} else if imgType == "png" {
		png.Encode(w, img)
	} else {
		w.Write(data)
	}
}

func (this *Server) ResizeImage(w http.ResponseWriter, fullpath string, width, height uint) {
	var (
		img     image.Image
		err     error
		imgType string
		file    *os.File
	)
	file, err = os.Open(fullpath)
	if err != nil {
		log.Error(err)
		return
	}
	img, imgType, err = image.Decode(file)
	if err != nil {
		log.Error(err)
		return
	}
	file.Close()
	img = resize.Resize(width, height, img, resize.Lanczos3)
	if imgType == "jpg" || imgType == "jpeg" {
		jpeg.Encode(w, img, nil)
	} else if imgType == "png" {
		png.Encode(w, img)
	} else {
		file.Seek(0, 0)
		io.Copy(w, file)
	}
}

func (this *Server) ParseSmallFile(filename string) (string, int64, int, error) {
	var (
		err    error
		offset int64
		length int
	)
	err = errors.New("unvalid small file")
	if len(filename) < 3 {
		return filename, -1, -1, err
	}
	if strings.Contains(filename, "/") {
		filename = filename[strings.LastIndex(filename, "/")+1:]
	}
	pos := strings.Split(filename, ",")
	if len(pos) < 3 {
		return filename, -1, -1, err
	}
	offset, err = strconv.ParseInt(pos[1], 10, 64)
	if err != nil {
		return filename, -1, -1, err
	}
	if length, err = strconv.Atoi(pos[2]); err != nil {
		return filename, offset, -1, err
	}
	if length > constDefine.CONST_SMALL_FILE_SIZE || offset < 0 {
		err = errors.New("invalid filesize or offset")
		return filename, -1, -1, err
	}
	return pos[0], offset, length, nil
}
