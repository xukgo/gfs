package core

//func InitConfig(supportGroupManage bool, group string) {
//	if supportGroupManage {
//		staticHandler = http.StripPrefix("/"+group+"/", http.FileServer(http.Dir(STORE_DIR)))
//	} else {
//		staticHandler = http.StripPrefix("/", http.FileServer(http.Dir(STORE_DIR)))
//	}
//	Singleton = NewServer()
//	Singleton.initComponent(false)
//}

//func Config() *configRepo.GloablConfig {
//	cnf := (*configRepo.GloablConfig)(atomic.LoadPointer(&ptr))
//	return cnf
//}
//func ParseConfig(filePath string) {
//	var (
//		data []byte
//	)
//	if filePath == "" {
//		data = []byte(strings.TrimSpace(constDefine.CONF_JSON_TEMPLATE))
//	} else {
//		file, err := os.Open(filePath)
//		if err != nil {
//			panic(fmt.Sprintln("open file path:", filePath, "error:", err))
//		}
//		defer file.Close()
//		FileName = filePath
//		data, err = ioutil.ReadAll(file)
//		if err != nil {
//			panic(fmt.Sprintln("file path:", filePath, " read all error:", err))
//		}
//	}
//	var c configRepo.GloablConfig
//	if err := json.Unmarshal(data, &c); err != nil {
//		panic(fmt.Sprintln("file path:", filePath, "json unmarshal error:", err))
//	}
//	log.Info(c)
//	atomic.StorePointer(&ptr, unsafe.Pointer(&c))
//	log.Info("config parse success")
//}
