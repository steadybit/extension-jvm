package jvmhttp

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/remote_jvm_connections"
	"github.com/steadybit/extension-kit/extutil"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
)

var (
	regexSanitize = regexp.MustCompile("[\n\t]")
)

func javaagent(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		if r.RemoteAddr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Debug().Msgf("Received request from %s", r.RemoteAddr)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleInternal(w, r.RemoteAddr, string(body))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleInternal(w http.ResponseWriter, remoteAddress string, body string) {
	log.Info().Msgf("Received request from %s with body %s", remoteAddress, body)

	tokens := strings.SplitN(regexSanitize.ReplaceAllString(body, "_"), "=", 2)
	if len(tokens) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	pid := extutil.ToInt32(tokens[0])
	jvmRemote := tokens[1]

	jvmRemoteSplitted := strings.SplitN(jvmRemote, ":", 2)
	var host string
	var port int
	if len(jvmRemoteSplitted) > 1 {
		addr, err := net.LookupIP(jvmRemoteSplitted[0])
		if err != nil {
			log.Warn().Err(err).Msgf("javaagent from unknown host")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		host = addr[0].String()
		port = extutil.ToInt(jvmRemoteSplitted[1])
	} else {
		splittedRemoteAddress := strings.Split(remoteAddress, ":")
		host = splittedRemoteAddress[0]
		port = extutil.ToInt(jvmRemote)
	}
	remote_jvm_connections.AddConnection(pid, host, port)
}
