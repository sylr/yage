# vi: ft=Dockerfile:

ARG GO_VERSION=1.16

FROM --platform=$BUILDPLATFORM golang:$GO_VERSION AS go

RUN apt-get update && apt-get dist-upgrade -y && apt-get install -y build-essential git

WORKDIR $GOPATH/src/sylr.dev/yage

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# -----------------------------------------------------------------------------

FROM --platform=$BUILDPLATFORM go AS builder

ARG TARGETPLATFORM

# Run a git command otherwise git describe in the Makefile could report a dirty git dir
RUN git diff --exit-code || true

RUN ["/bin/bash", "-c", "make build \
GOOS=$(cut -d '/' -f1 <<<\"$TARGETPLATFORM\") \
GOARCH=$(cut -d '/' -f2 <<<\"$TARGETPLATFORM\") \
GOARM=$(cut -d '/' -f3 <<<\"$TARGETPLATFORM\" | sed \"s/^v//\") \
GO_BUILD_TARGET=dist/$TARGETPLATFORM/yage"]

# -----------------------------------------------------------------------------

FROM scratch

ARG TARGETPLATFORM

WORKDIR /usr/local/bin

COPY --from=builder "/go/src/sylr.dev/yage/dist/$TARGETPLATFORM/yage" .

CMD ["/usr/local/bin/yage"]
