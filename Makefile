# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'


# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## tidy: format code and tidy modfile
.PHONY: tidy
tidy:
	go fmt ./...
	go mod tidy -v

## audit: run quality control checks
.PHONY: audit
audit:
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000,-ST1003 ./...
	go test -race -vet=off -coverprofile=coverage.out ./...
	go mod verify

## charttesting: Run Helm chart unit tests
.PHONY: charttesting
charttesting:
	for dir in charts/steadybit-extension-*; do \
    echo "Unit Testing $$dir"; \
    helm unittest $$dir; \
  done

## chartlint: Lint charts
.PHONY: chartlint
chartlint:
	ct lint --config chartTesting.yaml

# ==================================================================================== #
# BUILD
# ==================================================================================== #

## build: build the extension
.PHONY: build
build:
	go mod verify
	go build -o=./extension

## run: run the extension
.PHONY: run
run: tidy build
	./extension

## container: build the container image
.PHONY: container
container:
	#mvn clean package -DskipTests -f ./javaagents/pom.xml
	docker buildx build --build-arg ADDITIONAL_BUILD_PARAMS="-cover" -t extension-jvm:latest --output=type=docker .
	#docker buildx build --build-arg ADDITIONAL_BUILD_PARAMS="-cover" -t extension-jvm:latest --output=type=docker . -f Dockerfile-Debug

## java: build the java packages
.PHONY: java
java:
	mvn clean package -DskipTests -f ./javaagents/pom.xml

## javatest: run the java tests
.PHONY: javatest
javatest:
	mvn clean verify -f ./javaagents/pom.xml
