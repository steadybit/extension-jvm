//go:build linux

package common

import "github.com/rs/zerolog/log"
import "kernel.org/pub/linux/libs/security/libcap/cap"

func PrintCaps() {
	proc := cap.GetProc()
	log.Debug().Msgf("Capabilities: %s", proc.String())
}
