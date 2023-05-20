# syntax=docker/dockerfile:1

FROM alpine:3.18.0
WORKDIR /server
COPY build/pong .
ENTRYPOINT ["/server/pong"]
EXPOSE 80
