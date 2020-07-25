package orchestrator

import (
	"github.com/xukgo/gfs/configRepo"
	"github.com/xukgo/gfs/core"
	"github.com/xukgo/gsaber/utils/fileUtil"
)

func Start() error {
	configUrl := fileUtil.GetAbsUrl("conf/cfg.json")
	err := configRepo.InitRepo(configUrl)
	if err != nil {
		return err
	}
	err = core.Start(configRepo.GetSingleton())
	if err != nil {
		return err
	}
	return nil
}
