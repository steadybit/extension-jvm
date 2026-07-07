# Changelog

## v1.2.18

- chore(deps): bump actions/checkout from 6 to 7
- chore(deps): bump github.com/shirou/gopsutil/v4 from 4.26.5 to 4.26.6
- chore(deps): bump github.com/steadybit/action-kit/go/action_kit_commons
- chore(deps): bump github.com/steadybit/action-kit/go/action_kit_sdk
- chore(deps): bump github.com/steadybit/discovery-kit/go/discovery_kit_sdk
- chore(deps): bump github.com/steadybit/extension-kit
- chore(deps): bump golang.org/x/net from 0.55.0 to 0.56.0
- chore(deps): bump k8s.io/api from 0.36.1 to 0.36.2
- chore(deps): bump k8s.io/apimachinery from 0.36.1 to 0.36.2
- chore(deps): bump k8s.io/client-go from 0.36.1 to 0.36.2
- chore(deps): runc 1.4.3
- chore: add Claude Code workflows (#406)
- chore: silence SonarQube finding on secrets: inherit in Claude workflows
- feat: lower `oom_score_adj` on startup via extension-kit's `extruntime.AdjustOOMScoreAdj()` to avoid being killed by the node OOM killer. The extension sets it directly using the `cap_sys_resource` file capability (default `-998`, configurable via `STEADYBIT_EXTENSION_OOM_SCORE_ADJ`).
- feat: lower oom_score_adj on startup via extension-kit (#402)
- fix(chart): make javaagent jars readable across SELinux MCS boundaries on OpenShift (#409)
- fix(chart): on OpenShift, run the extension pod with an MCS-category-less SELinux level (`s0`) so target JVMs can read the mounted javaagent jars. Without this, SELinux denies the agent jar read (each namespace gets distinct MCS categories) and attacks fail with "connection not found". The SCC pins the level via `seLinuxContext: MustRunAs`; override via `podSecurityContext.seLinuxOptions` (mirrored into the SCC).
- fix: missing-circuit-breaker and missing-timeout accumulated wrong entries (#405)
- fix: propagate the underlying error when stopping an action fails, instead of returning a bare "Failed to stop action"
- fix: propagate underlying error when stopping an action fails (#408)

## v1.2.17

- chore(deps): bump github.com/shirou/gopsutil/v4 from 4.26.4 to 4.26.5
- chore(deps): bump github.com/steadybit/action-kit/go/action_kit_commons
- chore(deps): bump golang.org/x/net from 0.54.0 to 0.55.0
- chore(deps): bump golang.org/x/sys from 0.44.0 to 0.45.0
- chore(deps): bump golang.org/x/sys from 0.45.0 to 0.46.0
- chore: bump runc/crun and update trivyignore
- chore: update to go 1.26.4
- feat: add weekly auto patch-release workflow

## v1.2.16

- Support discovery group attribute via `STEADYBIT_EXTENSION_DISCOVERY_GROUP` env var (or `discovery.group` Helm value) — when set, the extension adds `steadybit.group=<value>` to every discovered target
- Update dependencies

## v1.2.15

- Bump Go to 1.26.3
- Update dependencies

## v1.2.14

- Bump Go to 1.25.9

## v1.2.13

- Update dependencies

## v1.2.12

- Use target query to narrow down attack targets
- Change Spring based attack labels
- Support if-none-match for the extension list endpoint
- feat(chart): split image.name into image.registry + image.name
- Support global.priorityClassName
- Update dependencies

## v1.2.11

- Update dependencies

## v1.2.10

- fix: regression on EKS not finding runc

## v1.2.9

- fix: prevent pickiung up JAVA_TOOL_OPTIONS when using crun

## v1.2.8

- Update dependencies

## v1.2.7

- Update dependencies

## v1.2.6

- Update dependencies

## v1.2.5

- Support Spring Path Patterns for HTTP Client attacks
- Update dependencies

## v1.2.4

- Update dependencies

## v1.2.3

- Updated dependencies

## v1.2.2

- Update dependencies
- Fix: JVM processes are lost in discovery when wall clock changes

## v1.2.1

- Update dependencies
## v1.2.1

- Update dependencies

## v1.2.0
- Breaking Change: Remove unreliable capturing of application context for spring boot applications
- Fix: more reliable discovery for jvm processes

## v1.1.11

- Set new `Technology` property in extension description
- Update dependencies (go 1.23)

## v1.1.10

- JVM excludes via vm arguments (like `steadybit.agent.disable-jvm-attachment`) are working again
- Option to validate user provided class and method name for "Java Method Delay" and" "Java Method Exception" attacks
- Align method parameter of "Controller Exception" and "Controller Delay" to "HTTP Client Status" and accept multiple values
- Change default value for "jitter" in all "Delay" attacks to false
- Fix graceful shutdown

## v1.1.8

- Update dependencies (go 1.22)
- Added namespace label to jvm-instance

## v1.1.7

- update dependencies

## v1.1.6

- update dependencies

## v1.1.5

- feat: add host.domainname attribute containing the host FQDN

## v1.1.4

- update dependencies

## v1.1.4

- update dependencies

## v1.1.3

- update dependencies

## v1.1.2

- update dependencies

## v1.1.1

- update dependencies

## v1.1.0

- Renamed application name to instance

## v1.0.13

- Removed jvm advice
- Fixed flaky spring discovery

## v1.0.12

- Update dependencies

## v1.0.11

- Enrich `application.name` to `container` targets
- Fixed `container-to-jvm` enrichment
- Update dependencies
- Prepared jvm advice

## v1.0.10

- Possibility to exclude attributes from discovery

## v1.0.9

- Improve process discovery
- Add enrichment rules for AWS attributes

## v1.0.8

- Improve process discovery
- Add enrichment rules for AWS attributes

## v1.0.7

- fix application discovery for rolling node deployments
- refactor spring and datasource discovery

## v1.0.6

- fix application name discovery

## v1.0.5

- update dependencies

## v1.0.4

- migration to new unified steadybit actionIds and targetTypes

## v1.0.3

- update dependencies

## v1.0.2

 - fix rpm dependencies

## v1.0.1

 - add linux package build

## v1.0.0

 - Initial release
