FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY agent/go.mod ./agent/go.mod
COPY agent/main.go ./agent/main.go
WORKDIR /src/agent
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /netvigia .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates iputils-ping snmp && rm -rf /var/lib/apt/lists/* \
 && useradd --system --create-home --home-dir /var/lib/netvigia --shell /usr/sbin/nologin netvigia \
 && mkdir -p /opt/netvigia/web /var/lib/netvigia && chown -R netvigia:netvigia /var/lib/netvigia
COPY --from=build /netvigia /usr/bin/netvigia
COPY index.html /opt/netvigia/web/index.html
ENV NETVIGIA_LISTEN=0.0.0.0:8787 NETVIGIA_DATA_DIR=/var/lib/netvigia NETVIGIA_POLL_MINUTES=60 NETVIGIA_WEB_DIR=/opt/netvigia/web
EXPOSE 8787
USER netvigia
ENTRYPOINT ["/usr/bin/netvigia"]
