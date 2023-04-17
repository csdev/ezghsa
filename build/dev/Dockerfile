FROM golang:1.19-alpine

# update this if different user and directory names are needed
ARG SERVICE=go

# should match host uid and gid
ARG UID=1000
ARG GID=1000

RUN apk add --no-cache gcc musl-dev tini

RUN apk add --no-cache --virtual .temp shadow \
    && groupadd --gid "$GID" "$SERVICE" \
    && useradd --uid "$UID" --gid "$GID" -m "$SERVICE" \
    && apk del .temp

# install deps and source code as non-root user, so that local development
# can also be done as non-root, while avoiding permissions issues on
# bind-mounted volumes (this works because $GOPATH is world-writable)
USER $SERVICE
WORKDIR /opt/$SERVICE
COPY --chown=$SERVICE go.mod go.sum ./
RUN go mod download && go mod verify

COPY --chown=$SERVICE . .

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/bin/sh"]
