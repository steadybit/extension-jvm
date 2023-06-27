# syntax=docker/dockerfile:1

##
## Build GO binary
##
FROM golang:1.20-bullseye AS build

ARG NAME
ARG VERSION
ARG REVISION
ARG ADDITIONAL_BUILD_PARAMS

WORKDIR /app

RUN apt-get -qq update \
    && apt-get -qq install -y --no-install-recommends build-essential
COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build \
    -ldflags="\
    -X 'github.com/steadybit/extension-kit/extbuild.ExtensionName=${NAME}' \
    -X 'github.com/steadybit/extension-kit/extbuild.Version=${VERSION}' \
    -X 'github.com/steadybit/extension-kit/extbuild.Revision=${REVISION}'" \
    -o ./extension \
    ${ADDITIONAL_BUILD_PARAMS}

##
## Runtime
##
FROM debian:bullseye-slim

ARG USERNAME=steadybit
ARG USER_UID=10000
ARG USER_GID=$USER_UID

RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME

RUN apt-get -qq update \
    && apt-get -qq install -y --no-install-recommends libcap2-bin runc \
    && apt-get -y autoremove \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /opt/steadybit/extension

COPY --from=build /app/extension /opt/steadybit/extension/extension
COPY javaagents/download/target/javaagent /opt/steadybit/extension/javaagent
RUN chown -R $USERNAME:$USERNAME /opt/steadybit/extension
RUN setcap "cap_setuid,cap_setgid,cap_sys_admin,cap_dac_override+eip" /opt/steadybit/extension/extension
USER $USERNAME

WORKDIR /opt/steadybit/extension


ENV STEADYBIT_EXTENSION_JAVA_AGENT_PATH=/opt/steadybit/extension/javaagent
EXPOSE 8085 8081

ENTRYPOINT ["/opt/steadybit/extension/extension"]
