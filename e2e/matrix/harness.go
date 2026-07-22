//go:build matrix

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package matrix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/moby/moby/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
)

const (
	actionPrefix   = "com.steadybit.extension_jvm."
	discoveredPath = "/" + actionPrefix + "jvm-instance/discovery/discovered-targets"
	agentPort      = "8080/tcp"
	extPort        = "8087/tcp"
)

func extImage() string {
	if v := os.Getenv("MATRIX_EXT_IMAGE"); v != "" {
		return v
	}
	return "extension-jvm:latest"
}

// ociRuntimeRoot is the runc state root the extension enters to attach. The default
// matches Docker Desktop and standard Linux docker; override for other runtimes.
func ociRuntimeRoot() string {
	if v := os.Getenv("MATRIX_OCIRUNTIME_ROOT"); v != "" {
		return v
	}
	return "/run/docker/runtime-runc/moby"
}

// Harness owns the containers for a single matrix cell and drives attacks against it.
type Harness struct {
	ctx       context.Context
	cell      Cell
	sampleTag string
	samplesFS string // path to testdata/samples

	net    *testcontainers.DockerNetwork
	stub   testcontainers.Container
	sample testcontainers.Container
	ext    testcontainers.Container

	sampleCID  string // container id of the sample, used to pick the right discovered target
	sampleBase string // http://host:portmapped
	extBase    string
}

// buildSampleImage builds the per-cell sample image via docker buildx (the build
// step is kept as an explicit exec so images can be reused across restarts).
func (h *Harness) buildSampleImage() error {
	var dir string
	args := []string{"buildx", "build", "--output=type=docker", "-t", h.sampleTag}
	if h.cell.SampleType == "plain" {
		dir = filepath.Join(h.samplesFS, "plainjava")
		args = append(args,
			"--build-arg", "BUILDER_IMAGE="+h.cell.Builder,
			"--build-arg", "RUNTIME_IMAGE="+h.cell.Runtime,
			"--build-arg", "COMPILER_RELEASE="+h.cell.Compiler)
	} else {
		dir = filepath.Join(h.samplesFS, "springboot")
		profiles := ""
		if h.cell.RestClient {
			profiles = "-Pwith-restclient"
		}
		args = append(args,
			"--build-arg", "BUILDER_IMAGE="+h.cell.Builder,
			"--build-arg", "RUNTIME_IMAGE="+h.cell.Runtime,
			"--build-arg", "BOOT_VERSION="+h.cell.Boot,
			"--build-arg", "COMPILER_RELEASE="+h.cell.Compiler,
			"--build-arg", "MVN_PROFILES="+profiles)
	}
	args = append(args, ".")
	cmd := exec.CommandContext(h.ctx, "docker", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("build %s: %w\n%s", h.sampleTag, err, tail(string(out), 30))
	}
	return nil
}

func (h *Harness) Start() error {
	h.sampleTag = "jvm-matrix-sample:" + h.cell.Name()
	if err := h.buildSampleImage(); err != nil {
		return err
	}
	net, err := network.New(h.ctx)
	if err != nil {
		return fmt.Errorf("network: %w", err)
	}
	h.net = net

	if h.cell.SampleType == "spring" {
		stub, err := testcontainers.GenericContainer(h.ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:          "traefik/whoami",
				Networks:       []string{net.Name},
				NetworkAliases: map[string][]string{net.Name: {"stub"}},
			},
			Started: true,
		})
		if err != nil {
			return fmt.Errorf("stub: %w", err)
		}
		h.stub = stub
	}
	return h.bringUp()
}

// bringUp starts the sample + extension and waits for the sample to be reachable and
// its target fully attached/enriched. Shared by Start and Restart.
func (h *Harness) bringUp() error {
	if err := h.startSample(); err != nil {
		return err
	}
	if err := h.startExt(); err != nil {
		return err
	}
	if err := h.waitReachable(); err != nil {
		return err
	}
	return h.waitEnriched()
}

func (h *Harness) startSample() error {
	env := map[string]string{}
	if h.cell.SampleType == "spring" {
		env["DOWNSTREAM_URL"] = "http://stub/"
	}
	c, err := testcontainers.GenericContainer(h.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          h.sampleTag,
			Env:            env,
			ExposedPorts:   []string{agentPort},
			Networks:       []string{h.net.Name},
			NetworkAliases: map[string][]string{h.net.Name: {"sample"}},
		},
		Started: true,
	})
	if err != nil {
		return fmt.Errorf("sample: %w", err)
	}
	h.sample = c
	h.sampleCID = c.GetContainerID()
	base, err := c.PortEndpoint(h.ctx, agentPort, "http")
	if err != nil {
		return err
	}
	h.sampleBase = base
	return nil
}

func (h *Harness) startExt() error {
	c, err := testcontainers.GenericContainer(h.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        extImage(),
			ExposedPorts: []string{extPort},
			Networks:     []string{h.net.Name},
			Env: map[string]string{
				"STEADYBIT_EXTENSION_RUNTIME":                           "docker",
				"STEADYBIT_EXTENSION_OCIRUNTIME_ROOT":                   ociRuntimeRoot(),
				"STEADYBIT_EXTENSION_MIN_PROCESS_AGE_BEFORE_ATTACHMENT": "5s",
				"STEADYBIT_LOG_LEVEL":                                   "INFO",
			},
			HostConfigModifier: func(hc *container.HostConfig) {
				hc.Privileged = true
				hc.PidMode = "host"
				hc.Binds = append(hc.Binds,
					"/var/run/docker.sock:/var/run/docker.sock",
					"/sys/fs/cgroup:/sys/fs/cgroup")
			},
		},
		Started: true,
	})
	if err != nil {
		return fmt.Errorf("extension: %w", err)
	}
	h.ext = c
	base, err := c.PortEndpoint(h.ctx, extPort, "http")
	if err != nil {
		return err
	}
	h.extBase = base
	return nil
}

func (h *Harness) waitReachable() error {
	path := "/mvc"
	if h.cell.SampleType == "plain" {
		path = "/health"
	}
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		if code, _ := h.probe(path); code == 200 {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("sample %s never became reachable", h.cell.Name())
}

// waitEnriched blocks until the extension has attached and fully enriched the
// sample's target. For Spring samples instance.type must contain spring-boot AND the
// actuator-JMX cycle must have populated the MVC mappings (a later cycle than the
// spring-boot flag) — firing an MVC/JDBC/HTTP-client attack before that yields
// "mappings not found". Attach + full enrichment typically takes ~60s.
func (h *Harness) waitEnriched() error {
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		if t := h.sampleTarget(); t != nil && h.enriched(t) {
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("target for %s never became fully enriched", h.cell.Name())
}

func (h *Harness) enriched(t *target) bool {
	hasType := func(v string) bool {
		for _, x := range t.Attributes["instance.type"] {
			if x == v {
				return true
			}
		}
		return false
	}
	if h.cell.SampleType == "plain" {
		return hasType("java")
	}
	return hasType("spring-boot") && len(t.Attributes["spring-instance.mvc-mapping"]) > 0
}

type target struct {
	ID         string              `json:"target"`
	Attributes map[string][]string `json:"attributes"`
}

// sampleTarget returns the discovered target that belongs to this cell's sample
// container. The extension runs with PidMode=host, so discovery may surface other
// JVMs on the host; matching on the sample's container id avoids acting on the wrong one.
func (h *Harness) sampleTarget() *target {
	var body struct {
		Targets []target `json:"targets"`
	}
	if err := h.extGet(discoveredPath, &body); err != nil {
		return nil
	}
	for i := range body.Targets {
		for _, cid := range body.Targets[i].Attributes["container.id.stripped"] {
			if cid == h.sampleCID {
				return &body.Targets[i]
			}
		}
	}
	return nil
}

// RunAttack executes one attack and classifies whether its effect was observable.
func (h *Harness) RunAttack(spec AttackSpec) AttackResult {
	res := AttackResult{Label: spec.Label, Mode: spec.Mode, Endpoint: spec.Endpoint}
	tgt := h.sampleTarget()
	if tgt == nil {
		res.Result, res.Detail = "error", "sample target not found"
		return res
	}
	res.Baseline = obs(h.probe(spec.Endpoint))

	exec := uuid.New().String()
	path := "/" + actionPrefix + spec.ActionID
	prepReq := map[string]any{
		"executionId": exec,
		"target":      map[string]any{"name": tgt.ID, "attributes": tgt.Attributes},
		"config":      spec.Config,
	}
	var prep actionResponse
	if err := h.extPostJSON(path+"/prepare", prepReq, &prep); err != nil {
		res.Result, res.Detail = "error", "prepare: "+err.Error()
		return res
	}
	if prep.Error != nil {
		res.Result, res.Detail = "error", "prepare: "+stringifyErr(prep.Error)
		return res
	}
	var start actionResponse
	if err := h.extPostJSON(path+"/start", map[string]any{"executionId": exec, "state": prep.State}, &start); err != nil {
		res.Result, res.Detail = "error", "start: "+err.Error()
		return res
	}
	if start.Error != nil {
		res.Result, res.Detail = "error", stringifyErr(start.Error)
		return res
	}
	state := start.State
	if state == nil {
		state = prep.State
	}
	dCode, dT := h.pollDuring(spec)
	res.During = obs(dCode, dT)

	var stop actionResponse
	_ = h.extPostJSON(path+"/stop", map[string]any{"executionId": exec, "state": state}, &stop)
	time.Sleep(3 * time.Second)
	aCode, aT := h.probe(spec.Endpoint)
	res.After = obs(aCode, aT)

	if spec.effectObserved(dCode, dT) && spec.recovered(aCode, aT) {
		res.Result = "pass"
	} else {
		res.Result = "fail"
	}
	return res
}

// actionResponse is the shape returned by the ActionKit prepare/start/stop endpoints
// that this suite consumes.
type actionResponse struct {
	State map[string]any `json:"state"`
	Error map[string]any `json:"error"`
}

func obs(code int, t float64) []any { return []any{code, round(t)} }

// Restart recreates the sample + extension so each attack can run against a fresh
// JVM (running many attacks on one JVM destabilizes ByteBuddy instrumentation).
func (h *Harness) Restart() error {
	if h.ext != nil {
		_ = h.ext.Terminate(h.ctx)
	}
	if h.sample != nil {
		_ = h.sample.Terminate(h.ctx)
	}
	return h.bringUp()
}

func (h *Harness) Teardown() {
	for _, c := range []testcontainers.Container{h.ext, h.sample, h.stub} {
		if c != nil {
			_ = c.Terminate(h.ctx)
		}
	}
	if h.net != nil {
		_ = h.net.Remove(h.ctx)
	}
}

// ---- helpers ----

// pollDuring repeatedly probes the endpoint while an attack is active, returning as
// soon as the expected effect appears (ByteBuddy instrumentation activates
// asynchronously after /start, so a single fixed-delay probe is racy). It returns the
// last observation if the effect never appears within the window.
func (h *Harness) pollDuring(spec AttackSpec) (int, float64) {
	deadline := time.Now().Add(10 * time.Second)
	for {
		code, t := h.probe(spec.Endpoint)
		if spec.effectObserved(code, t) || !time.Now().Before(deadline) {
			return code, t
		}
		time.Sleep(1 * time.Second)
	}
}

func (h *Harness) probe(path string) (int, float64) {
	client := &http.Client{Timeout: 30 * time.Second}
	start := time.Now()
	resp, err := client.Get(h.sampleBase + path)
	if err != nil {
		return -1, -1
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, time.Since(start).Seconds()
}

func (h *Harness) extGet(path string, out any) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(h.extBase + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(out)
	_, _ = io.Copy(io.Discard, resp.Body) // drain so the keep-alive connection is reused
	return err
}

func (h *Harness) extPostJSON(path string, body, out any) error {
	b, _ := json.Marshal(body)
	client := &http.Client{Timeout: 40 * time.Second}
	resp, err := client.Post(h.extBase+path, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, path, tail(string(data), 3))
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}

func stringifyErr(m map[string]any) string {
	if d, ok := m["detail"]; ok {
		return fmt.Sprint(d)
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

func round(f float64) float64 { return math.Round(f*1000) / 1000 }
