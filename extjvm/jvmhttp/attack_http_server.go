package jvmhttp

import (
	"github.com/rs/zerolog/log"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
)

type Status struct {
	Started    bool
	Stopped    bool
	Failure    string
	ConfigJson string
	Pid        int32
	port       int
	listener   *net.Listener
}

var (
	attackStatus sync.Map // port -> AttackStatus
)

func StartAttackHttpServer(pid int32, configJson string) int {
	serverMuxProbes := http.NewServeMux()
	serverMuxProbes.Handle("/", handler(handleGetConfig))
	serverMuxProbes.Handle("/started", handler(handleStarted))
	serverMuxProbes.Handle("/stopped", handler(handleStopped))
	serverMuxProbes.Handle("/failed", handler(handleFailed))

	log.Info().Msg("Starting HTTP server for java agent attack communication ")
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Err(err).Msg("Failed to start HTTP server.")
		return 0
	}
	port := listener.Addr().(*net.TCPAddr).Port
	log.Info().Msgf("Listening on port %d", port)

	saveAttackStatus(port, Status{
		Started:    false,
		Stopped:    false,
		Failure:    "",
		Pid:        pid,
		port:       port,
		listener:   &listener,
		ConfigJson: configJson,
	})

	go func() {
		_ = http.Serve(listener, serverMuxProbes)
	}()
	return port
}

func StopAttackHttpServer(pid int32) {
	status := GetAttackStatus(pid)
	if status.listener != nil {
		if err := (*status.listener).Close(); err != nil {
			log.Err(err).Msgf("Failed to close listener on port %d.", status.port)
			return
		}
	}
	attackStatus.Delete(status.port)
}

func handler(handler func(Status, string, http.ResponseWriter) Status) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		port, err := getPort(request)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		status := loadAttackStatus(port)

		defer func(body io.ReadCloser) {
			if err := body.Close(); err != nil {
				log.Debug().Msg("Failed to close body.")
			}
		}(request.Body)
		body, err := io.ReadAll(request.Body)
		if err != nil {
			log.Err(err).Msg("Failed to read body.")
		}

		newStatus := handler(status, string(body), writer)
		saveAttackStatus(port, newStatus)
	}
}

func handleFailed(status Status, body string, _ http.ResponseWriter) Status {
	log.Error().Msgf("Attack failed: %s for pid %d.", body, status.Pid)
	status.Failure = body
	status.Stopped = true
	return status
}

func handleStopped(status Status, _ string, _ http.ResponseWriter) Status {
	log.Info().Msgf("Attack stopped for pid %d.", status.Pid)
	status.Stopped = true
	return status
}

func handleStarted(status Status, _ string, _ http.ResponseWriter) Status {
	log.Info().Msgf("Attack started for pid %d.", status.Pid)
	status.Started = true
	return status
}

func handleGetConfig(status Status, _ string, writer http.ResponseWriter) Status {
	writer.Header().Set("Content-Type", "application/json")
	if _, err := writer.Write([]byte(status.ConfigJson)); err != nil {
		log.Error().Msg("Failed to write config.")
	} else {
		log.Debug().Msgf("Attack Config delivered for pid %d: %s", status.Pid, status.ConfigJson)
	}
	return status
}

func getPort(request *http.Request) (int, error) {
	ctx := request.Context()
	srvAddr := ctx.Value(http.LocalAddrContextKey).(net.Addr)
	_, port, err := net.SplitHostPort(srvAddr.String())
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(port)
}

func loadAttackStatus(port int) Status {
	status, ok := attackStatus.Load(port)
	if !ok {
		return Status{port: port}
	}
	return status.(Status)
}

func saveAttackStatus(port int, status Status) {
	attackStatus.Store(port, status)
}

func GetAttackStatus(pid int32) Status {
	var status Status
	attackStatus.Range(func(key, value interface{}) bool {
		if value.(Status).Pid == pid {
			status = value.(Status)
			return false
		}
		return true
	})
	return status
}
