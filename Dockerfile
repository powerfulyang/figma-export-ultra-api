# syntax=docker/dockerfile:1

ARG GO_VERSION=1.21

FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /app
RUN apk add --no-cache git ca-certificates && update-ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN go generate ./ent
RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.20
RUN addgroup -S app && adduser -S app -G app && apk add --no-cache ca-certificates && update-ca-certificates
WORKDIR /app
COPY --from=build /out/server /usr/local/bin/server
USER app:app
ENV APP_ENV=prod SERVER_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/server"]

