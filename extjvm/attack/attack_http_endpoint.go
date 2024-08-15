package attack

import (
	"github.com/rs/zerolog/log"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
)

type Status struct {
	Started       bool
	Stopped       bool
	AdviceApplied string
	Failure       string
	ConfigJson    string
	Pid           int32
	Listener      *net.Listener
	Port          int
}

var (
	attackStatus sync.Map // port -> AttackStatus
)

func StartAttackEndpoint(pid int32, configJson string) int {
	serverMuxProbes := http.NewServeMux()
	serverMuxProbes.Handle("/", http.HandlerFunc(getConfig))
	serverMuxProbes.Handle("/started", http.HandlerFunc(started))
	serverMuxProbes.Handle("/stopped", http.HandlerFunc(stopped))
	serverMuxProbes.Handle("/failed", http.HandlerFunc(failed))

	log.Info().Msg("Starting HTTP server for java agent attack communication ")
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Err(err).Msg("Failed to start HTTP server.")
		return 0
	}
	port := listener.Addr().(*net.TCPAddr).Port
	log.Info().Msgf("Listening on port %d", port)
	saveAttackStatus(strconv.Itoa(port), Status{
		Started:       false,
		Stopped:       false,
		AdviceApplied: "UNKNOWN",
		Failure:       "",
		Pid:           pid,
		Port:          port,
		Listener:      &listener,
		ConfigJson:    configJson,
	})
	go func() {
		_ = http.Serve(listener, serverMuxProbes)
	}()
	return port
}

func StopAttackEndpoint(pid int32) {
	status := GetAttackStatus(pid)
	if status.Listener != nil {
		err := (*status.Listener).Close()
		if err != nil {
			log.Err(err).Msg("Failed to close listener.")
			return
		}
	}
	attackStatus.Delete(strconv.Itoa(status.Port))
}
func failed(writer http.ResponseWriter, request *http.Request) {
	body := request.Body
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			log.Error().Msg("Failed to close body.")
		}
	}(body)
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		log.Err(err).Msg("Failed to read body.")
	}
	bodyString := string(bodyBytes)
	port, err := getPort(request)
	if err != nil {
		log.Error().Msg("Failed to get port.")
		return
	}
	status := loadAttackStatus(port)
	status.Failure = bodyString
	status.Stopped = true
	saveAttackStatus(port, status)
	log.Error().Msgf("Attack failed: %s for pid %d.", bodyString, status.Pid)
	writer.WriteHeader(http.StatusOK)
}

func getPort(request *http.Request) (string, error) {
	ctx := request.Context()
	srvAddr := ctx.Value(http.LocalAddrContextKey).(net.Addr)
	_, port, err := net.SplitHostPort(srvAddr.String())
	return port, err
}

func loadAttackStatus(port string) Status {
	status, ok := attackStatus.Load(port)
	if !ok {
		return Status{}
	}
	return status.(Status)
}

func saveAttackStatus(port string, status Status) {
	attackStatus.Store(port, status)
}

func stopped(_ http.ResponseWriter, request *http.Request) {
	port, err := getPort(request)
	if err != nil {
		log.Error().Msg("Failed to get port.")
		return
	}
	status := loadAttackStatus(port)
	status.Stopped = true
	saveAttackStatus(port, status)
	log.Info().Msgf("Attack stopped for pid %d.", status.Pid)
}

func started(_ http.ResponseWriter, request *http.Request) {
	body := request.Body
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			log.Error().Msg("Failed to close body.")
		}
	}(body)
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		log.Err(err).Msg("Failed to read body.")
	}
	bodyString := string(bodyBytes)

	port, err := getPort(request)
	if err != nil {
		log.Error().Msg("Failed to get port.")
		return
	}
	status := loadAttackStatus(port)
	status.Started = true
	status.AdviceApplied = bodyString
	saveAttackStatus(port, status)
	log.Info().Msgf("Attack started for pid %d. Advice status '%s'", status.Pid, status.AdviceApplied)
}

func getConfig(writer http.ResponseWriter, request *http.Request) {
	port, err := getPort(request)
	if err != nil {
		log.Error().Msg("Failed to get port.")
		return
	}
	status := loadAttackStatus(port)
	configJson := status.ConfigJson
	writer.Header().Set("Content-Type", "application/json")
	_, err = writer.Write([]byte(configJson))
	if err != nil {
		log.Error().Msg("Failed to write config.")
		return
	}
	log.Debug().Msgf("Attack Config delivered for pid %d.", status.Pid)
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
