# vi: ft=Dockerfile:

FROM scratch

ARG VERSION
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

WORKDIR /usr/local/bin

COPY dist/yage-${VERSION}-${TARGETOS}-${TARGETARCH}${TARGETVARIANT} yage

CMD ["/usr/local/bin/yage"]
