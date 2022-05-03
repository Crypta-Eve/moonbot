# syntax=docker/dockerfile:1

FROM golang:1.18-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY ore_variants.json ./
COPY *.go ./

RUN CGO_ENABLED=0 go build -o moonbot .

FROM alpine

COPY --from=build /app/moonbot /app/moonbot

ENTRYPOINT ["/app/moonbot"]
