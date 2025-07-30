FROM node:22-alpine AS build-web
WORKDIR /app
COPY . .
RUN npm install
RUN npm run build

FROM golang:1.24-bookworm as builder-base
RUN apt-get update && apt-get install -y build-essential curl
RUN curl https://install.duckdb.org | sh

FROM builder-base AS build
WORKDIR /go/src/kea
COPY . .
COPY --from=build-web /app/dist /go/src/kea/dist

ENV CGO_ENABLED=1
RUN go build -v -o /go/bin/kea ./main.go

FROM debian:12-slim
WORKDIR /app
COPY --from=build /go/bin/kea /app/kea

EXPOSE 8080
CMD ["/app/kea"]
