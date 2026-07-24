//go:build matrix

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package matrix

// mode describes how an attack's effect is verified against the sample endpoint.
const (
	modeDelay = "delay" // endpoint latency must rise to ~delayMillis while active, recover after
	modeError = "error" // endpoint must return >=500 while active, recover to 200 after
)

const (
	delayMillis    = 2000
	durationMillis = 30000

	// delayThresholdSecs is the latency a delay attack must add to count as observed.
	// Derived from delayMillis (75% of it) so it tracks the configured delay instead of
	// being an independent magic number.
	delayThresholdSecs = float64(delayMillis) / 1000 * 0.75
	recoveredSecs      = 1.0
)

// AttackSpec is one attack invocation with the config that actually takes effect
// (see e2e/matrix/README.md for the non-obvious config requirements this encodes:
// explicit erroneousCallRate, concrete httpMethods, actuator-backed discovery).
type AttackSpec struct {
	Label    string // display label incl. client variant, e.g. "spring-httpclient-delay[resttemplate]"
	ActionID string // steadybit action id suffix (without the extension prefix)
	Mode     string // modeDelay | modeError
	Endpoint string // sample endpoint used to observe the effect
	Config   map[string]any
}

// effectObserved reports whether a probe (status, latency) shows the attack's effect.
// For delay attacks the latency must rise above the baseline by the threshold, so a
// merely slow endpoint (high baseline, or a transient spike near it) isn't mistaken for
// an injected delay.
func (s AttackSpec) effectObserved(code int, latencySecs, baselineSecs float64) bool {
	if s.Mode == modeDelay {
		return latencySecs-baselineSecs >= delayThresholdSecs
	}
	return code >= 500
}

// recovered reports whether a probe shows the endpoint back to normal after stop.
// The status must be 200 in both modes: probe returns the sentinel (-1, -1) on a
// failed request, so checking latency alone would score a crashed/unreachable sample
// as "recovered".
func (s AttackSpec) recovered(code int, latencySecs float64) bool {
	if s.Mode == modeDelay {
		return code == 200 && latencySecs < recoveredSecs
	}
	return code == 200
}

// springAttacks returns the attacks applicable to a Spring Boot sample.
// withRestClient must reflect whether the sample image was built with the RestClient
// endpoint (Boot >= 3.2 / the with-restclient profile); otherwise the restclient
// variants would target a non-existent endpoint and be scored as false failures.
func springAttacks(withRestClient bool) []AttackSpec {
	specs := []AttackSpec{
		{"spring-mvc-delay", "spring-mvc-delay-attack", modeDelay, "/mvc",
			map[string]any{"pattern": "/mvc", "methods": []string{"*"}, "delay": delayMillis, "duration": durationMillis, "delayJitter": false}},
		{"spring-mvc-exception", "spring-mvc-exception-attack", modeError, "/mvc",
			map[string]any{"pattern": "/mvc", "methods": []string{"*"}, "duration": durationMillis, "erroneousCallRate": 100}},
		{"spring-jdbctemplate-delay", "spring-jdbctemplate-delay-attack", modeDelay, "/jdbc",
			map[string]any{"operations": "*", "delay": delayMillis, "duration": durationMillis, "delayJitter": false, "jdbcUrl": "*"}},
		{"spring-jdbctemplate-exception", "spring-jdbctemplate-exception-attack", modeError, "/jdbc",
			map[string]any{"operations": "*", "duration": durationMillis, "jdbcUrl": "*", "erroneousCallRate": 100}},
		{"java-method-delay", "java-method-delay-attack", modeDelay, "/mvc",
			map[string]any{"className": "com.steadybit.matrix.ComputeService", "methodName": "compute", "delay": delayMillis, "duration": durationMillis, "delayJitter": false, "validate": true}},
		{"java-method-exception", "java-method-exception-attack", modeError, "/mvc",
			map[string]any{"className": "com.steadybit.matrix.ComputeService", "methodName": "compute", "duration": durationMillis, "validate": true, "erroneousCallRate": 100}},
	}
	// One row per HTTP client variant the sample exposes. RestClient only exists on Boot >= 3.2.
	clients := []struct{ name, endpoint string }{
		{"resttemplate", "/http/resttemplate"},
		{"webclient", "/http/webclient"},
	}
	if withRestClient {
		clients = append(clients, struct{ name, endpoint string }{"restclient", "/http/restclient"})
	}
	for _, c := range clients {
		specs = append(specs,
			AttackSpec{"spring-httpclient-delay[" + c.name + "]", "spring-httpclient-delay-attack", modeDelay, c.endpoint,
				map[string]any{"delay": delayMillis, "duration": durationMillis, "delayJitter": false, "httpMethods": []string{"GET"}, "hostAddress": "*", "urlPath": "/**"}},
			AttackSpec{"spring-httpclient-status[" + c.name + "]", "spring-httpclient-status-attack", modeError, c.endpoint,
				map[string]any{"duration": durationMillis, "httpMethods": []string{"GET"}, "hostAddress": "*", "urlPath": "/**", "failureCauses": []string{"HTTP_500"}, "erroneousCallRate": 100}},
		)
	}
	return specs
}

// plainAttacks returns the attacks applicable to the non-Spring Java sample.
func plainAttacks() []AttackSpec {
	const cls, mth, ep = "com.steadybit.matrix.WorkService", "work", "/work"
	return []AttackSpec{
		{"java-method-delay", "java-method-delay-attack", modeDelay, ep,
			map[string]any{"className": cls, "methodName": mth, "delay": delayMillis, "duration": durationMillis, "delayJitter": false, "validate": true}},
		{"java-method-exception", "java-method-exception-attack", modeError, ep,
			map[string]any{"className": cls, "methodName": mth, "duration": durationMillis, "validate": true, "erroneousCallRate": 100}},
	}
}
