package configRepo

import jsoniter "github.com/json-iterator/go"

type Repo struct {
	Addr                 string   `json:"addr"`
	Peers                []string `json:"peers"`
	EnableHttps          bool     `json:"enable_https"`
	Group                string   `json:"group"`
	RenameFile           bool     `json:"rename_file"`
	ShowDir              bool     `json:"show_dir"`
	Extensions           []string `json:"extensions"`
	RefreshInterval      int      `json:"refresh_interval"`
	EnableWebUpload      bool     `json:"enable_web_upload"`
	DownloadDomain       string   `json:"download_domain"`
	EnableCustomPath     bool     `json:"enable_custom_path"`
	Scenes               []string `json:"scenes"`
	DefaultScene         string   `json:"default_scene"`
	DownloadUseToken     bool     `json:"download_use_token"`
	DownloadTokenExpire  int      `json:"download_token_expire"`
	QueueSize            int      `json:"queue_size"`
	AutoRepair           bool     `json:"auto_repair"`
	Host                 string   `json:"host"`
	FileSumArithmetic    string   `json:"file_sum_arithmetic"`
	PeerId               string   `json:"peer_id"`
	SupportGroupManage   bool     `json:"support_group_manage"`
	AdminIps             []string `json:"admin_ips"`
	EnableMergeSmallFile bool     `json:"enable_merge_small_file"`
	EnableMigrate        bool     `json:"enable_migrate"`
	EnableDistinctFile   bool     `json:"enable_distinct_file"`
	EnableCrossOrigin    bool     `json:"enable_cross_origin"`
	AuthUrl              string   `json:"auth_url"`
	EnableDownloadAuth   bool     `json:"enable_download_auth"`
	DefaultDownload      bool     `json:"default_download"`
	EnableTus            bool     `json:"enable_tus"`
	SyncTimeout          int64    `json:"sync_timeout"`
	ConnectTimeout       bool     `json:"connect_timeout"`
	ReadTimeout          int      `json:"read_timeout"`
	WriteTimeout         int      `json:"write_timeout"`
	IdleTimeout          int      `json:"idle_timeout"`
	ReadHeaderTimeout    int      `json:"read_header_timeout"`
	SyncWorker           int      `json:"sync_worker"`
	UploadWorker         int      `json:"upload_worker"`
	UploadQueueSize      int      `json:"upload_queue_size"`
	RetryCount           int      `json:"retry_count"`
	SyncDelay            int64    `json:"sync_delay"`
	WatchChanSize        int      `json:"watch_chan_size"`
}

func (this *Repo) ToJson() []byte {
	gson, _ := jsoniter.ConfigCompatibleWithStandardLibrary.Marshal(this)
	return gson
}
func (this *Repo) FillWithJson(data []byte) error {
	return jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal(data, this)
}
func (this *Repo) GetAddr() string {
	return this.Addr
}
func (this *Repo) GetPeers() []string {
	return this.Peers
}
func (this *Repo) GetEnableHttps() bool {
	return this.EnableHttps
}
func (this *Repo) GetGroup() string {
	return this.Group
}
func (this *Repo) GetRenameFile() bool {
	return this.RenameFile
}
func (this *Repo) GetShowDir() bool {
	return this.ShowDir
}
func (this *Repo) GetExtensions() []string {
	return this.Extensions
}
func (this *Repo) GetRefreshInterval() int {
	return this.RefreshInterval
}
func (this *Repo) GetEnableWebUpload() bool {
	return this.EnableWebUpload
}
func (this *Repo) GetDownloadDomain() string {
	return this.DownloadDomain
}
func (this *Repo) SetDownloadDomain(val string) {
	this.DownloadDomain = val
}
func (this *Repo) GetEnableCustomPath() bool {
	return this.EnableCustomPath
}
func (this *Repo) GetScenes() []string {
	return this.Scenes
}
func (this *Repo) GetDefaultScene() string {
	return this.DefaultScene
}
func (this *Repo) GetDownloadUseToken() bool {
	return this.DownloadUseToken
}
func (this *Repo) GetDownloadTokenExpire() int {
	return this.DownloadTokenExpire
}
func (this *Repo) GetQueueSize() int {
	return this.QueueSize
}
func (this *Repo) GetAutoRepair() bool {
	return this.AutoRepair
}
func (this *Repo) GetHost() string {
	return this.Host
}
func (this *Repo) GetFileSumArithmetic() string {
	return this.FileSumArithmetic
}
func (this *Repo) GetPeerId() string {
	return this.PeerId
}
func (this *Repo) GetSupportGroupManage() bool {
	return this.SupportGroupManage
}
func (this *Repo) GetAdminIps() []string {
	return this.AdminIps
}
func (this *Repo) GetEnableMergeSmallFile() bool {
	return this.EnableMergeSmallFile
}
func (this *Repo) GetEnableMigrate() bool {
	return this.EnableMigrate
}
func (this *Repo) GetEnableDistinctFile() bool {
	return this.EnableDistinctFile
}
func (this *Repo) GetEnableCrossOrigin() bool {
	return this.EnableCrossOrigin
}
func (this *Repo) GetAuthUrl() string {
	return this.AuthUrl
}
func (this *Repo) GetEnableDownloadAuth() bool {
	return this.EnableDownloadAuth
}
func (this *Repo) GetDefaultDownload() bool {
	return this.DefaultDownload
}
func (this *Repo) GetEnableTus() bool {
	return this.EnableTus
}
func (this *Repo) GetSyncTimeout() int64 {
	return this.SyncTimeout
}
func (this *Repo) GetConnectTimeout() bool {
	return this.ConnectTimeout
}
func (this *Repo) GetReadTimeout() int {
	return this.ReadTimeout
}
func (this *Repo) GetWriteTimeout() int {
	return this.WriteTimeout
}
func (this *Repo) GetIdleTimeout() int {
	return this.IdleTimeout
}
func (this *Repo) GetReadHeaderTimeout() int {
	return this.ReadHeaderTimeout
}
func (this *Repo) GetSyncWorker() int {
	return this.SyncWorker
}
func (this *Repo) GetUploadWorker() int {
	return this.UploadWorker
}
func (this *Repo) GetUploadQueueSize() int {
	return this.UploadQueueSize
}
func (this *Repo) GetRetryCount() int {
	return this.RetryCount
}
func (this *Repo) GetSyncDelay() int64 {
	return this.SyncDelay
}
func (this *Repo) GetWatchChanSize() int {
	return this.WatchChanSize
}
