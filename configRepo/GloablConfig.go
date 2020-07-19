package configRepo

type GloablConfig struct {
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
