package core

//
//http.HandleFunc(fmt.Sprintf("%s/reload", groupRoute), this.Reload)
//func (this *Server) Reload(w http.ResponseWriter, r *http.Request) {
//	var (
//		err     error
//		data    []byte
//		cfg     configRepo.GloablConfig
//		action  string
//		cfgjson string
//		result  model.JsonResult
//	)
//	result.Status = "fail"
//	r.ParseForm()
//	if !this.IsPeer(r) {
//		w.Write([]byte(this.GetClusterNotPermitMessage(r)))
//		return
//	}
//	cfgjson = r.FormValue("cfg")
//	action = r.FormValue("action")
//	_ = cfgjson
//	if action == "get" {
//		result.Data = configRepo.GetSingleton()
//		result.Status = "ok"
//		w.Write([]byte(this.util.JsonEncodePretty(result)))
//		return
//	}
//	if action == "set" {
//		if cfgjson == "" {
//			result.Message = "(error)parameter cfg(json) require"
//			w.Write([]byte(this.util.JsonEncodePretty(result)))
//			return
//		}
//		if err = json.Unmarshal([]byte(cfgjson), &cfg); err != nil {
//			log.Error(err)
//			result.Message = err.Error()
//			w.Write([]byte(this.util.JsonEncodePretty(result)))
//			return
//		}
//		result.Status = "ok"
//		cfgjson = this.util.JsonEncodePretty(cfg)
//		this.util.WriteFile(this.confRepo.GetConfFileName(), cfgjson)
//		w.Write([]byte(this.util.JsonEncodePretty(result)))
//		return
//	}
//	if action == "reload" {
//		if data, err = ioutil.ReadFile(this.confRepo.GetConfFileName()); err != nil {
//			result.Message = err.Error()
//			w.Write([]byte(this.util.JsonEncodePretty(result)))
//			return
//		}
//		if err = json.Unmarshal(data, &cfg); err != nil {
//			result.Message = err.Error()
//			w.Write([]byte(this.util.JsonEncodePretty(result)))
//			return
//		}
//		configRepo.InitRepo(this.confRepo.GetConfFileName())
//		this.initComponent(true)
//		result.Status = "ok"
//		w.Write([]byte(this.util.JsonEncodePretty(result)))
//		return
//	}
//	if action == "" {
//		w.Write([]byte("(error)action support set(json) get reload"))
//	}
//}
