ARG BUILD_FROM=ghcr.io/home-assistant/amd64-base:latest
FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS builder

ARG TARGETARCH
ARG TARGETVARIANT

RUN apk add --no-cache ca-certificates git tzdata

WORKDIR /src/GoSungrow
COPY go.mod go.sum main.go ./
COPY cmd ./cmd
COPY cmdHassio ./cmdHassio
COPY cmdModbus ./cmdModbus
COPY defaults ./defaults
COPY iSolarCloud ./iSolarCloud
COPY tablib ./tablib

RUN go mod download

RUN set -eux; \
    export GOOS=linux; \
    case "${TARGETARCH}" in \
      amd64) export GOARCH=amd64 ;; \
      arm64) export GOARCH=arm64 ;; \
      arm) export GOARCH=arm; export GOARM="${TARGETVARIANT#v}"; [ -n "${GOARM}" ] || export GOARM=7 ;; \
      "") export GOARCH="$(go env GOARCH)" ;; \
      *) echo "unsupported TARGETARCH: ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/GoSungrow .

FROM ${BUILD_FROM}

ARG BUILD_ARCH
ARG BUILD_DESCRIPTION="Sungrow iSolarCloud MQTT bridge for Home Assistant"
ARG BUILD_NAME="GoSungrow"
ARG BUILD_VERSION

LABEL \
  io.hass.name="${BUILD_NAME}" \
  io.hass.description="${BUILD_DESCRIPTION}" \
  io.hass.arch="${BUILD_ARCH}" \
  io.hass.type="addon" \
  io.hass.version="${BUILD_VERSION}" \
  org.opencontainers.image.source="https://github.com/roth-andreas/gosungrow-home-assistant"

ENV HOME=/data \
    GOSUNGROW_CONFIG=/data/.GoSungrow/config.json \
    GOSUNGROW_ASSET_DIR=/opt/gosungrow/assets

COPY --from=builder /out/GoSungrow /usr/local/bin/GoSungrow
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY addon/gosungrow/run.sh /run.sh
COPY addon/gosungrow/assets /opt/gosungrow/assets

RUN chmod a+x /run.sh

WORKDIR /data
CMD ["/run.sh"]
