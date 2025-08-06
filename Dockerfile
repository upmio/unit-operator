# Build the unit-operator binary
FROM golang:1.23.6 AS builder

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
COPY cmd/main.go cmd/main.go
COPY api/ api/
COPY pkg/  pkg/
COPY .git .git

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o dist/unit-operator \
    -ldflags "-s -w"  \
    -ldflags "-X  'github.com/upmio/unit-operator/pkg/vars.GITCOMMIT=$(git rev-parse --short HEAD)'  -X 'github.com/upmio/unit-operator/pkg/vars.VERSION=$(git describe --abbrev=0 2>/dev/null)'  -X 'github.com/upmio/unit-operator/pkg/vars.BUILDTIME=$(date +'%Y-%m-%dT%H:%M:%S')'  -X 'github.com/upmio/unit-operator/pkg/vars.GITBRANCH=$(git rev-parse --abbrev-ref HEAD)'  -X 'github.com/upmio/unit-operator/pkg/vars.GOVERSION=$(go version | grep -o  'go[0-9].[0-9].*')'" \
    cmd/main.go

# Use distroless as minimal base image to package the unit-operator binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM rockylinux/rockylinux:9.5.20241118

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
    epel-release

WORKDIR /

COPY --from=builder /workspace/dist/unit-operator /usr/local/bin/unit-operator
RUN chmod -R 755 /usr/local/bin/unit-operator
USER 65532:65532

ENTRYPOINT ["/usr/local/bin/unit-operator"]
