package jvmhttp

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"strings"
)

func logHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		doLog(r.Body)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

type LogMessage struct {
	Level      string `json:"level"`
	Msg        string `json:"msg"`
	Stacktrace string `json:"stacktrace"`
	Pid        string `json:"pid"`
}

func doLog(body io.ReadCloser) {
	decoder := json.NewDecoder(body)
	var message LogMessage
	err := decoder.Decode(&message)
	if err != nil {
		log.Err(err).Msg("Failed to decode request body.")
		return
	}

	var sb strings.Builder
	var logLevel string
	var pid string

	if message.Level != "" {
		logLevel = message.Level
	} else {
		log.Error().Msgf("No log level provided: %+v", message)
		return
	}

	if message.Pid != "" {
		pid = message.Pid
	} else {
		log.Error().Msgf("No pid provided: %+v", message)
		return
	}

	log.Trace().Msgf("Received log entry from PID %s: %+v", pid, message)

	if message.Msg != "" {
		sb.WriteString(message.Msg)
	}

	if message.Stacktrace != "" {
		if message.Msg != "" {
			sb.WriteString("\n")
		}
		sb.WriteString(message.Stacktrace)
	}

	if strings.EqualFold(logLevel, "ERROR") {
		log.Error().Msgf("(PID %s) - %s", message.Pid, sb.String())
	} else if strings.EqualFold(logLevel, "WARN") {
		log.Warn().Msgf("(PID %s) - %s", message.Pid, sb.String())
	} else if strings.EqualFold(logLevel, "INFO") {
		log.Info().Msgf("(PID %s) - %s", message.Pid, sb.String())
	} else if strings.EqualFold(logLevel, "DEBUG") {
		log.Debug().Msgf("(PID %s) - %s", message.Pid, sb.String())
	} else if strings.EqualFold(logLevel, "TRACE") {
		log.Trace().Msgf("(PID %s) - %s", message.Pid, sb.String())
	} else {
		log.Error().Msgf("Unknown log level: %s", logLevel)
	}
}
