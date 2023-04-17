FROM golang:1.19-alpine AS builder

ARG SERVICE=go

WORKDIR /opt/$SERVICE
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN mkdir "/opt/$SERVICE/bin" \
    && go build -o "/opt/$SERVICE/bin" -v ./...

FROM alpine:latest

ARG SERVICE=go
ARG UID=1000
ARG GID=1000

RUN apk add --no-cache tini

RUN apk add --no-cache --virtual .temp shadow \
    && groupadd --gid "$GID" "$SERVICE" \
    && useradd --uid "$UID" --gid "$GID" -m "$SERVICE" \
    && apk del .temp

WORKDIR /opt/$SERVICE/bin
COPY --from=builder /opt/$SERVICE/bin .

USER $SERVICE
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/bin/sh"]
