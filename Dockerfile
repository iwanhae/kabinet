FROM node:22-alpine AS build-web
WORKDIR /app
COPY . .
RUN npm install
RUN npm run build

FROM golang:1.24-alpine AS build
WORKDIR /go/src/kew
COPY . .
COPY --from=build-web /app/dist /go/src/kew/dist

ENV CGO_ENABLED=0
RUN go build -v -o /go/bin/kew ./main.go

FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=build /go/bin/kew /app/kew

EXPOSE 8080
CMD ["/app/kew"]