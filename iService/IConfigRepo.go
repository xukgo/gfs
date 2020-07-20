package iService

type IConfigRepo interface {
	GetAddr() string
	GetPeers() []string
	GetEnableHttps() bool
	GetGroup() string
	GetRenameFile() bool
	GetShowDir() bool
	GetExtensions() []string
	GetRefreshInterval() int
	GetEnableWebUpload() bool
	GetDownloadDomain() string
	SetDownloadDomain(string)
	GetEnableCustomPath() bool
	GetScenes() []string
	GetDefaultScene() string
	GetDownloadUseToken() bool
	GetDownloadTokenExpire() int
	GetQueueSize() int
	GetAutoRepair() bool
	GetHost() string
	GetFileSumArithmetic() string
	GetPeerId() string
	GetSupportGroupManage() bool
	GetAdminIps() []string
	GetEnableMergeSmallFile() bool
	GetEnableMigrate() bool
	GetEnableDistinctFile() bool
	GetEnableCrossOrigin() bool
	GetAuthUrl() string
	GetEnableDownloadAuth() bool
	GetDefaultDownload() bool
	GetEnableTus() bool
	GetSyncTimeout() int64
	GetConnectTimeout() bool
	GetReadTimeout() int
	GetWriteTimeout() int
	GetIdleTimeout() int
	GetReadHeaderTimeout() int
	GetSyncWorker() int
	GetUploadWorker() int
	GetUploadQueueSize() int
	GetRetryCount() int
	GetSyncDelay() int64
	GetWatchChanSize() int
}
