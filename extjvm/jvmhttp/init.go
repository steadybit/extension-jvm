package jvmhttp

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
)

func Start(port uint16) {
	serverMuxProbes := http.NewServeMux()
	serverMuxProbes.Handle("/javaagent", http.HandlerFunc(javaagent))
	serverMuxProbes.Handle("/log", http.HandlerFunc(logHandler))

	go func() {
		log.Info().Msgf("Starting HTTP server for java agent communication on port %d", port)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), serverMuxProbes); err != nil {
			log.Err(err).Msg("Failed to start HTTP server.")
			return
		}
	}()
}
