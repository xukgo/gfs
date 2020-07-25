package main

import (
	"flag"
	"fmt"
	log "github.com/sjqzhang/seelog"
	"github.com/xukgo/gfs/orchestrator"
	"os"
	"path/filepath"
	"strings"
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

	checkExecDir()
	err := orchestrator.Start()
	if err != nil {
		os.Exit(-1)
	}
}

func checkExecDir() {
	appDir, e1 := filepath.Abs(filepath.Dir(os.Args[0]))
	curDir, e2 := filepath.Abs(".")
	if e1 == nil && e2 == nil && appDir != curDir && !strings.Contains(appDir, "go-build") {
		msg := fmt.Sprintf("please change directory to '%s' start fileserver\n", appDir)
		msg = msg + fmt.Sprintf("请切换到 '%s' 目录启动 fileserver ", appDir)
		log.Warn(msg)
		fmt.Println(msg)
		os.Exit(1)
	}
}
