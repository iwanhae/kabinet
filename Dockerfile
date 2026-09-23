FROM node:22-alpine AS build-web
WORKDIR /app
COPY . .
RUN npm install
RUN npm run build

FROM golang:1.27-bookworm AS builder-base
ARG TARGETARCH
ARG DUCKDB_VERSION=1.5.5
RUN apt-get update \
    && apt-get install -y --no-install-recommends build-essential ca-certificates curl unzip \
    && rm -rf /var/lib/apt/lists/*
RUN case "${TARGETARCH}" in \
    amd64|arm64) duckdb_arch="${TARGETARCH}" ;; \
    *) echo "unsupported architecture: ${TARGETARCH}" >&2; exit 1 ;; \
    esac \
    && mkdir /tmp/duckdb \
    && curl -fsSL -o /tmp/duckdb/cli.zip \
    "https://github.com/duckdb/duckdb/releases/download/v${DUCKDB_VERSION}/duckdb_cli-linux-${duckdb_arch}.zip" \
    && curl -fsSL -o /tmp/duckdb/lib.zip \
    "https://github.com/duckdb/duckdb/releases/download/v${DUCKDB_VERSION}/libduckdb-linux-${duckdb_arch}.zip" \
    && unzip -q /tmp/duckdb/cli.zip -d /tmp/duckdb/cli \
    && unzip -q /tmp/duckdb/lib.zip -d /tmp/duckdb/lib \
    && install -m 0755 /tmp/duckdb/cli/duckdb /usr/local/bin/duckdb \
    && install -m 0644 /tmp/duckdb/lib/libduckdb.so /usr/local/lib/libduckdb.so \
    && rm -rf /tmp/duckdb

FROM builder-base AS build
WORKDIR /go/src/kabinet
COPY . .
COPY --from=build-web /app/dist /go/src/kabinet/cmd/server/dist

ENV CGO_ENABLED=1
ENV CGO_LDFLAGS="-L/usr/local/lib -lduckdb"
RUN go build -v -tags=duckdb_use_lib -o /go/bin/server ./cmd/server/main.go
RUN go build -v -tags=duckdb_use_lib -o /go/bin/compactor ./cmd/compactor/main.go

FROM debian:12
WORKDIR /
COPY --from=build /go/bin/server /usr/local/bin/server
COPY --from=build /go/bin/compactor /usr/local/bin/compactor
COPY --from=builder-base /usr/local/bin/duckdb /usr/local/bin/duckdb
COPY --from=builder-base /usr/local/lib/libduckdb.so /usr/local/lib/libduckdb.so
RUN ldconfig

EXPOSE 8080
VOLUME [ "/data" ]
CMD ["server"]
