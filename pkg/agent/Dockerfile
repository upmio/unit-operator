# Build the unit-agent binary
FROM golang:1.24.6 AS builder

ARG TARGETOS
ARG TARGETARCH

ENV GO111MODULE on

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY api/ api/
COPY pkg pkg/
COPY .git .git


# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o dist/unit-agent \
    -ldflags "-s -w" \
    -ldflags "-X 'github.com/upmio/unit-operator/pkg/agent/version.GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)' -X 'github.com/upmio/unit-operator/pkg/agent/version.GIT_COMMIT=$(git rev-parse HEAD)' -X 'github.com/upmio/unit-operator/pkg/agent/version.BUILD_TIME=$(date '+%Y-%m-%d %H:%M:%S')' -X 'github.com/upmio/unit-operator/pkg/agent/version.GO_VERSION=$(go version | grep -o  'go[0-9].[0-9].*')'" \
    pkg/agent/main.go


# Use distroless as minimal base image to package the unit-operator binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM rockylinux/rockylinux:9.6.20250531

# set timezone
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime \
  && echo "$TZ" > /etc/timezone

# install common tools
RUN set -eux; \
  dnf install -y \
    procps-ng \
    net-tools \
    telnet \
    epel-release; \
  groupadd --system --gid 1001 "unit-agent"; \
  useradd --system --uid 1001 --gid 1001 -m -c "Default Application User" "unit-agent";

WORKDIR /home/unit-agent
COPY --from=builder /workspace/dist/unit-agent /usr/local/bin/unit-agent
COPY --from=builder /workspace/pkg/agent/etc/config.toml /opt/unit-agent/config.toml

CMD ["unit-agent","daemon","-f","/opt/unit-agent/config.toml"]

USER unit-agent
EXPOSE 2214
