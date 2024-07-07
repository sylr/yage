# vi: ft=Dockerfile:

ARG GO_VERSION=1.22

FROM --platform=$BUILDPLATFORM golang:$GO_VERSION AS builder

RUN --mount=type=cache,target=/var/cache/apt \
    apt-get update && apt-get dist-upgrade -y && apt-get install -y build-essential git

WORKDIR $GOPATH/src/sylr.dev/yage

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download

ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

# Switch shell to bash
SHELL ["bash", "-c"] 

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,target=. \
    git diff --exit-code || true; \
    make build \
        GOOS=${TARGETOS} \
        GOARCH=${TARGETARCH} \
        GOARM=${TARGETVARIANT/v/} \
        GO_BUILD_TARGET=/tmp/dist/${TARGETPLATFORM}/yage \
        GO_BUILD_DIR=/tmp/dist/${TARGETPLATFORM} \
        GO_BUILD_FLAGS_TARGET=/tmp/.go-build-flags 

# -----------------------------------------------------------------------------

FROM scratch

ARG TARGETPLATFORM

WORKDIR /usr/local/bin

COPY --from=builder "/tmp/dist/$TARGETPLATFORM/yage" .

CMD ["/usr/local/bin/yage"]
