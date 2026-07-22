# Attack Support Matrix (BM-13107)

On-demand suite that determines **which attacks work on which Java LTS × Spring Boot version**
by actually attaching the extension to a real sample JVM, running each attack, and observing the
effect. It regenerates [`RESULTS.md`](./RESULTS.md) and `results.json`.

It lives in the main module but every file is gated behind the `matrix` build tag, so it is
excluded from normal builds and `make test`. It is image- and time-heavy — intended for
`workflow_dispatch` in CI or a deliberate local run, not the normal test cycle.

## Run

```bash
# from the repo root — builds extension-jvm:latest, then runs the full grid
make matrix

# or directly, for a subset (from the repo root)
go test -tags matrix -timeout 3h -run TestSupportMatrix -v ./e2e/matrix/...
```

Prerequisites: a working Docker daemon and the extension image (`make container` builds
`extension-jvm:latest`).

### Environment knobs

| Var | Default | Meaning |
|---|---|---|
| `MATRIX_CELLS` | all | comma-separated substrings selecting a subset, e.g. `boot4.1.0,plainjava-java21` |
| `MATRIX_ISOLATION` | `per-attack` | `per-attack` = fresh JVM per attack (correct, slow); `per-cell` = one JVM per cell (fast, less reliable) |
| `MATRIX_EXT_IMAGE` | `extension-jvm:latest` | extension image tag to attach with |

## How it works

Per cell, the harness builds the sample image (`../testdata/samples/{springboot,plainjava}`),
then starts — on a shared Docker network — a downstream stub (`traefik/whoami`), the sample, and
the extension. The extension replicates the production DaemonSet attach path: `privileged`,
`PidMode=host`, the Docker socket and cgroup fs bind-mounted, `STEADYBIT_EXTENSION_RUNTIME=docker`
and the default OCI runc root. It then polls discovery until the JVM is attached and (for Spring)
fully enriched, and drives each attack over the ActionKit HTTP API, asserting the observable
effect on a sample endpoint (added latency for delay attacks, `>=500` for error attacks) and
recovery after stop.

## Non-obvious requirements baked into this suite

These were discovered empirically; changing them silently breaks results:

- **Samples must include `spring-boot-starter-actuator`.** The extension reads beans/mappings
  from actuator's JMX MBeans; without it Spring is detected but every bean/mapping is empty, so
  MVC and HTTP-client attacks have nothing to target.
- **Wait for full enrichment.** `instance.type` only reaches `spring-boot` ~60s after container
  start (attach + Spring discovery cycle). Firing earlier misses the Spring attributes.
- **`erroneousCallRate: 100` must be sent explicitly** for all exception/status attacks — the UI
  default is not applied server-side, so omitting it yields a 0% (no-op) attack.
- **`httpMethods` must be concrete** (e.g. `["GET"]`), not `["*"]` — the agent matcher does an
  exact per-method compare with no wildcard (`hostAddress`/`urlPath` *do* wildcard).
- **Isolate attacks.** Running many attacks against one JVM destabilizes ByteBuddy instrumentation
  and can crash the extension; the default `per-attack` mode uses a fresh JVM each time.

## Known open item

`spring-httpclient-status` loads and configures correctly but does not inject a failure status
(extension reports `Advice status 'UNKNOWN'`), even though the delay advice on the same client
works. Tracked as a candidate bug in the status advice — see BM-13107.
