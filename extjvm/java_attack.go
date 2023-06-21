package extjvm

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/attachment"
	"github.com/steadybit/extension-jvm/extjvm/attack"
	"github.com/steadybit/extension-jvm/extjvm/common"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"sync"
	"time"
)

var (
	attackJavaJavaAgentJar = common.GetJarPath("attack-java-javaagent.jar")

	attackStartTimeout = 10 * time.Second
)

func Prepare(jvm *jvm.JavaVm, configJson string) (string, int) {
	attackEndpointPort := attack.StartAttackEndpoint(jvm.Pid, configJson)
	// The callback URL is used to send the attack results back to the agent.
	host := attachment.GetAttachment(jvm).GetAgentHost()
	callbackUrl := fmt.Sprintf("http://%s:%d", host, attackEndpointPort)
	log.Trace().Msgf("Callback URL: %s", callbackUrl)
	return callbackUrl, attackEndpointPort
}
func Start(jvm *jvm.JavaVm, callbackUrl string) error {
	success, err := LoadAgentPlugin(jvm, attackJavaJavaAgentJar, callbackUrl)
	if err != nil {
		return err
	}
	if !success {
		log.Warn().Msg("Failed to load attack plugin.")
	}

	// Wait for the attack to start.
	timeout := attackStartTimeout
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for {
			status := attack.GetAttackStatus(jvm.Pid)
			if status.Started {
				wg.Done()
				return
			} else if status.Failure != "" {
				log.Error().Msgf("Failed to start attack: %s", status.Failure)
				wg.Done()
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()
	if utils.WaitTimeout(&wg, timeout) {
		return errors.New("Timed out waiting for Java Attack instrumentation after " + timeout.String())
	}

	return nil
}

func Stop(jvm *jvm.JavaVm) bool {
	attack.StopAttackEndpoint(jvm.Pid)
	success, err := UnloadAgentPlugin(jvm, attackJavaJavaAgentJar)
	if err != nil {
		log.Error().Msgf("Failed to unload attack plugin: %s", err)
		return false
	}
	if !success {
		log.Warn().Msg("Failed to unload attack plugin.")
		return false
	}
	return true
}
