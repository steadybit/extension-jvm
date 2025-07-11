# Changelog

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
