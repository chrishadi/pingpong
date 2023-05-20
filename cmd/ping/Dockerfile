# syntax=docker/dockerfile:1

FROM alpine:3.18.0
WORKDIR /app
COPY build/ping .
ENTRYPOINT ["/app/ping"]
