package controller

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/attachment/remote_jvm_connections"
	"github.com/steadybit/extension-kit/extutil"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
)

func javaagent(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "PUT":
		remoteAddress := r.RemoteAddr
		var status uint16
		if remoteAddress == "" {
			status = 500
		} else {
			log.Debug().Msgf("Received request from %s", remoteAddress)
			body, err := io.ReadAll(r.Body)
			if err != nil {
				log.Err(err).Msg("Failed to read request body.")
				w.WriteHeader(400)
				return
			}
			status, err = handleInternal(r.RemoteAddr, string(body))
		}
		w.WriteHeader(int(status))
	default:
		_, err := fmt.Fprintf(w, "Sorry, only POST methods are supported.")
		if err != nil {
			log.Err(err).Msg("Failed to write response.")
			return
		}
	}
}

func handleInternal(remoteAddress string, body string) (uint16, error) {
	compile := regexp.MustCompile("[\n\n\t]")
	bodySanitized := compile.ReplaceAllString(body, "_")
	bodySplitted := strings.SplitN(bodySanitized, "=", 2)
	if len(bodySplitted) == 2 {
		pid := extutil.ToInt32(bodySplitted[0])
		jvmRemote := bodySplitted[1]
		jvmRemoteSplitted := strings.SplitN(jvmRemote, ":", 2)
		var host string
		var port int
		if len(jvmRemoteSplitted) > 1 {
			addr, err := net.LookupIP(jvmRemoteSplitted[0])
			if err != nil {
				return 400, errors.New("unknown host")
			}
			host = addr[0].String()
			port = extutil.ToInt(jvmRemoteSplitted[1])
		} else {
			host = remoteAddress
			port = extutil.ToInt(jvmRemote)
		}
		remote_jvm_connections.AddConnection(pid, host, port)
	} else {
		return 400, errors.New("invalid request body")
	}
	return 200, nil
}
