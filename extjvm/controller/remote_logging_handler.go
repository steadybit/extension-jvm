package controller

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"strings"
)

func logHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":

		doLog(r.Body)
		w.WriteHeader(200)
	default:
		_, err := fmt.Fprintf(w, "Sorry, only POST methods are supported.")
		if err != nil {
			log.Err(err).Msg("Failed to write response.")
			return
		}
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
	var json LogMessage
	err := decoder.Decode(&json)
	if err != nil {
		log.Err(err).Msg("Failed to decode request body.")
		return
	}

	var sb strings.Builder
	var logLevel string
	var pid string

	if json.Level != "" {
		logLevel = json.Level
	} else {
		log.Error().Msgf("No log level provided: %+v", json)
		return
	}

	if json.Pid != "" {
		pid = json.Pid
	} else {
		log.Error().Msgf("No pid provided: %+v", json)
		return
	}

	log.Trace().Msgf("Received log entry from PID %s: %+v", pid, json)

	if json.Msg != "" {
		sb.WriteString(json.Msg)
	}

	if json.Stacktrace != "" {
		if json.Msg != "" {
			sb.WriteString("\n")
		}
		sb.WriteString(json.Stacktrace)
	}

	if strings.EqualFold(logLevel, "ERROR") {
		log.Error().Msgf("(PID %s) - %s", json.Pid, sb.String())
	} else if strings.EqualFold(logLevel, "WARN") {
		log.Warn().Msgf("(PID %s) - %s", json.Pid, sb.String())
	} else if strings.EqualFold(logLevel, "INFO") {
		log.Info().Msgf("(PID %s) - %s", json.Pid, sb.String())
	} else if strings.EqualFold(logLevel, "DEBUG") {
		log.Debug().Msgf("(PID %s) - %s", json.Pid, sb.String())
	} else if strings.EqualFold(logLevel, "TRACE") {
		log.Trace().Msgf("(PID %s) - %s", json.Pid, sb.String())
	} else {
		log.Error().Msgf("Unknown log level: %s", logLevel)
	}
}
