package main

import (
	"flag"
	"fmt"
	"github.com/xukgo/gfs/configRepo"
	"github.com/xukgo/gfs/core"
	"github.com/xukgo/gsaber/utils/fileUtil"
	"os"
)

var (
	VERSION    string
	BUILD_TIME string
	GO_VERSION string
)

func main() {
	v := flag.Bool("v", false, "display version")
	flag.Parse()
	if *v {
		fmt.Printf("%s\n%s\n%s\n", VERSION, BUILD_TIME, GO_VERSION)
		os.Exit(0)
	}

	configUrl := fileUtil.GetAbsUrl("conf/cfg.json")
	err := configRepo.InitRepo(configUrl)
	if err != nil {
		return
	}

	core.InitConfig(configRepo.GetSingleton().GetSupportGroupManage(), configRepo.GetSingleton().Group)
	core.Singleton.Start()
}
