# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.20-bookworm AS build

ARG BUILD_WITH_COVERAGE
ARG BUILD_SNAPSHOT=true

WORKDIR /app

RUN echo 'deb [trusted=yes] https://repo.goreleaser.com/apt/ /' > /etc/apt/sources.list.d/goreleaser.list \
    && apt-get -qq update \
    && apt-get -qq install -y --no-install-recommends build-essential libcap2-bin goreleaser

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN goreleaser build --snapshot="${BUILD_SNAPSHOT}" --single-target -o extension \
    && setcap "cap_setuid,cap_setgid,cap_sys_admin,cap_dac_override+eip" ./extension

##
## Runtime
##
FROM debian:bookworm-slim

LABEL "steadybit.com.discovery-disabled"="true"

ARG USERNAME=steadybit
ARG USER_UID=10000
ARG USER_GID=$USER_UID

RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME

RUN apt-get -qq update \
    && apt-get -qq install -y --no-install-recommends runc libcap2-bin procps \
    && apt-get -y autoremove \
    && rm -rf /var/lib/apt/lists/*

USER $USERNAME

WORKDIR /

COPY --from=build /app/extension /extension
COPY --from=build /app/licenses /licenses
COPY javaagents/download/target/javaagent /javaagent


ENV STEADYBIT_EXTENSION_JAVA_AGENT_PATH=/javaagent
EXPOSE 8085 8081

ENTRYPOINT ["/extension"]
