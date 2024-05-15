package extjvm

import (
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/steadybit/extension-jvm/extjvm/jvmhttp"
	"time"
)

var (
	attackStartTimeout = 10 * time.Second
)

func start(jvm *jvm.JavaVm, pluginJar, callbackUrl string) error {
	if err := loadAgentPlugin(jvm, pluginJar, callbackUrl); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), attackStartTimeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for Java Attack instrumentation after %s", attackStartTimeout)
		case <-time.After(100 * time.Millisecond):
			status := jvmhttp.GetAttackStatus(jvm.Pid)
			if status.Started {
				return nil
			} else if status.Failure != "" {
				return errors.New("Failed to start attack: " + status.Failure)
			}
		}
	}
}

func stop(jvm *jvm.JavaVm, pluginJar string) bool {
	jvmhttp.StopAttackHttpServer(jvm.Pid)
	success, err := unloadAgentPlugin(jvm, pluginJar)
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
