//go:build matrix

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

// Package matrix contains the on-demand attack-support-matrix suite for BM-13107.
//
// It is gated behind the `matrix` build tag so it never runs during `make test`.
// Run it with:
//
//	cd e2e/matrix && go test -tags matrix -timeout 3h -run TestSupportMatrix -v
//
// Prerequisites: a running Docker daemon and the extension image available as
// `extension-jvm:latest` (override with MATRIX_EXT_IMAGE), built via `make container`.
//
// Environment knobs:
//   - MATRIX_CELLS      comma-separated substrings to select a subset of cells (default: all)
//   - MATRIX_ISOLATION  "per-attack" (default, fresh JVM per attack) or "per-cell" (faster, one JVM)
//   - MATRIX_EXT_IMAGE  extension image tag (default extension-jvm:latest)
package matrix

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSupportMatrix(t *testing.T) {
	samplesFS := samplesDir(t)
	perAttack := os.Getenv("MATRIX_ISOLATION") != "per-cell"
	filter := splitCSV(os.Getenv("MATRIX_CELLS"))

	var results []CellResult
	for _, cell := range Cells() {
		if !matches(cell.Name(), filter) {
			continue
		}
		results = append(results, runCell(t, samplesFS, cell, perAttack))
	}

	if err := WriteJSON(filepath.Join(reportDir(), "results.json"), results); err != nil {
		t.Fatalf("write results.json: %v", err)
	}
	if err := WriteMarkdown(filepath.Join(reportDir(), "RESULTS.md"), results); err != nil {
		t.Fatalf("write RESULTS.md: %v", err)
	}
	t.Logf("wrote RESULTS.md and results.json (%d cells)", len(results))
}

func runCell(t *testing.T, samplesFS string, cell Cell, perAttack bool) CellResult {
	t.Helper()
	cr := CellResult{Cell: cell.Name(), SampleType: cell.SampleType, Boot: cell.Boot, Java: cell.Java}
	ctx := context.Background()

	t.Run(cell.Name(), func(t *testing.T) {
		h := &Harness{ctx: ctx, cell: cell, samplesFS: samplesFS}
		if err := h.Start(); err != nil {
			cr.Error = err.Error()
			t.Errorf("cell setup: %v", err)
			h.Teardown()
			return
		}
		defer h.Teardown()

		attacks := cell.Attacks()
		for i, spec := range attacks {
			if perAttack && i > 0 {
				if err := h.Restart(); err != nil {
					cr.Attacks = append(cr.Attacks, AttackResult{Label: spec.Label, Result: "error", Detail: "restart: " + err.Error()})
					continue
				}
			}
			r := h.RunAttack(spec)
			t.Logf("%-42s %s (base=%v during=%v after=%v) %s", r.Label, r.Result, r.Baseline, r.During, r.After, r.Detail)
			cr.Attacks = append(cr.Attacks, r)
		}
	})
	return cr
}

func samplesDir(t *testing.T) string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve caller path")
	}
	// this file: e2e/matrix/matrix_test.go -> samples at e2e/testdata/samples
	return filepath.Join(filepath.Dir(thisFile), "..", "testdata", "samples")
}

// reportDir is the directory where RESULTS.md / results.json are written (e2e/matrix).
func reportDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Dir(thisFile)
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func matches(name string, filter []string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, f := range filter {
		if strings.Contains(name, f) {
			return true
		}
	}
	return false
}
