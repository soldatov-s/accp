FROM library/golang:1.15-alpine
RUN apk update && apk --no-cache add bash git make && \
    rm -rf /var/cache/apk/*

ARG CGO_ENABLED=0
ENV APP_DIR $GOPATH/src/github.com/soldatov-s/accp
WORKDIR $APP_DIR

# Compile the binary and statically link
COPY . .
RUN make build-stable && \
    ln -s $APP_DIR/config.yml /etc/accp/config.yml

FROM scratch

COPY --from=0 /go/bin/accp /usr/bin/accp
COPY --from=0 /etc/accp /etc/accp

USER 1000

# Set the entrypoint
ENTRYPOINT ["accp"]
