package main

import (
	"flag"
	"fmt"
	"github.com/xukgo/gfs/core"
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
	core.Singleton.Start()
}
