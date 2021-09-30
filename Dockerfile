# syntax=docker/dockerfile:1

FROM golang:1.17-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o moonbot .

FROM scratch

COPY --from=build /app/moonbot /app/moonbot

ENTRYPOINT ["/app/moonbot"]