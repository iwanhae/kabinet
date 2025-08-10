FROM node:22-alpine AS build-web
WORKDIR /app
COPY . .
RUN npm install
RUN npm run build

FROM golang:1.24-bookworm as builder-base
RUN apt-get update && apt-get install -y build-essential curl
RUN curl https://install.duckdb.org | sh

FROM builder-base AS build
WORKDIR /go/src/kabinet
COPY . .
COPY --from=build-web /app/dist /go/src/kabinet/dist

ENV CGO_ENABLED=1
RUN go build -v -o /go/bin/kabinet ./main.go

FROM debian:12-slim
WORKDIR /
COPY --from=build /go/bin/kabinet /app/kabinet

EXPOSE 8080
VOLUME [ "/data" ]
CMD ["/app/kabinet"]
