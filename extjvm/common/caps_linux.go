//go:build linux

package common

import (
	"github.com/rs/zerolog/log"
	"os"
	"os/exec"
	"strconv"
)

func PrintCaps() {
	if !log.Debug().Enabled() {
		return
	}

	out, err := exec.Command("getpcaps", strconv.Itoa(os.Getpid())).CombinedOutput()
	if err != nil {
		log.Debug().Err(err).Msgf("failed to get own capabilities")
	} else {
		log.Debug().Msgf("capabilities: %s", string(out))
	}
}
