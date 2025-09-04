package jvm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type javaagentHttpServer struct {
	server      *http.Server
	connections *jvmConnections
	port        int
}

func (b *javaagentHttpServer) listen(address string) {
	mux := http.NewServeMux()
	mux.Handle("/javaagent", http.HandlerFunc(b.javaagentHandler))
	mux.Handle("/log", http.HandlerFunc(logHandler))

	b.server = &http.Server{Handler: mux}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Err(err).Msg("Failed to start HTTP server.")
		return
	}
	b.port = listener.Addr().(*net.TCPAddr).Port
	log.Info().Msgf("Listen for javaagent HTTP connections on port %d", b.port)

	go func(server *http.Server, listener net.Listener) {
		if err = server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			log.Err(err).Msg("Failed to start HTTP server.")
		}
	}(b.server, listener)
}

func (b *javaagentHttpServer) shutdown() {
	if b.server == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := b.server.Shutdown(ctx); err != nil {
		log.Err(err).Msg("Failed to shutdown HTTP server.")
	}
}

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
	var message LogMessage
	if err := json.NewDecoder(body).Decode(&message); err != nil {
		log.Err(err).Msg("Failed to decode request body.")
		return
	}

	var sb strings.Builder
	if message.Msg != "" {
		sb.WriteString(message.Msg)
	}

	if message.Stacktrace != "" {
		if message.Msg != "" {
			sb.WriteString("\n")
		}
		sb.WriteString(message.Stacktrace)
	}

	var level zerolog.Level
	if strings.EqualFold(message.Level, "ERROR") {
		level = zerolog.ErrorLevel
	} else if strings.EqualFold(message.Level, "WARN") {
		level = zerolog.WarnLevel
	} else if strings.EqualFold(message.Level, "INFO") {
		level = zerolog.InfoLevel
	} else if strings.EqualFold(message.Level, "DEBUG") {
		level = zerolog.DebugLevel
	} else if strings.EqualFold(message.Level, "TRACE") {
		level = zerolog.TraceLevel
	} else {
		level = zerolog.InfoLevel
	}

	log.WithLevel(level).Msgf("(PID %s) - %s", message.Pid, sb.String())
}

func (b *javaagentHttpServer) javaagentHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		if r.RemoteAddr == "" {
			w.WriteHeader(http.StatusInternalServerError)
		} else if body, err := io.ReadAll(r.Body); err == nil {
			if err := b.doJavaagent(r.RemoteAddr, string(body)); err == nil {
				return
			}
		}
		w.WriteHeader(http.StatusBadRequest)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (b *javaagentHttpServer) doJavaagent(remoteAddress string, body string) error {
	log.Trace().Msgf("Received request from %s with body %s", remoteAddress, body)

	tokens := strings.SplitN(body, "=", 2)
	if len(tokens) != 2 {
		return errors.New("invalid body")
	}

	pid, err := strconv.Atoi(tokens[0])
	if err != nil {
		return errors.New("invalid pid")
	}

	jvmRemote := strings.SplitN(tokens[1], ":", 2)
	var portStr string
	var host string
	if len(jvmRemote) > 1 {
		portStr = jvmRemote[1]
		host, err = lookupFirstAddress(jvmRemote[0])
		if err != nil {
			return errors.New("unknown host")
		}
	} else {
		portStr = jvmRemote[0]
		host, _, err = net.SplitHostPort(remoteAddress)
		if err != nil {
			return errors.New("invalid remote address")
		}
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return errors.New("invalid port")
	}

	b.connections.addConnection(int32(pid), fmt.Sprintf("%s:%d", host, port))
	return nil
}

func lookupFirstAddress(host string) (string, error) {
	r, err := net.LookupIP(host)
	if err != nil {
		return "", err
	}
	return r[0].String(), nil
}
