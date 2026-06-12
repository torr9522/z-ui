package job

import (
	"x-ui/logger"
	"x-ui/web/service"
)

type CheckXrayRunningJob struct {
	xrayService service.XrayService

	checkTime int
}

func NewCheckXrayRunningJob() *CheckXrayRunningJob {
	return new(CheckXrayRunningJob)
}

func (j *CheckXrayRunningJob) Run() {
	if j.xrayService.IsXrayRunning() {
		if err := j.xrayService.CheckRuntimeHealth(); err != nil {
			logger.Warning("runtime api health check failed:", err)
		} else {
			j.checkTime = 0
			return
		}
	}
	if err := j.xrayService.RecoverRuntime(); err == nil {
		j.checkTime = 0
		return
	}
	j.checkTime++
	if j.checkTime < 2 {
		return
	}
	j.xrayService.SetToNeedRestart()
}
