FROM library/golang:1.15-alpine
RUN apk update && apk --no-cache add bash git make && \
	rm -rf /var/cache/apk/*

ARG CGO_ENABLED=0
ARG APP_NAME
ARG PACKAGE
ENV APP_DIR $GOPATH/src/$PACKAGE
WORKDIR $APP_DIR

# Compile the binary and statically link
COPY . .
RUN make build-stable && \
	cd /go/bin/; ln -s /go/bin/$APP_NAME /go/bin/service && \
	mkdir -p /etc/accp && \
	cp $APP_DIR/config.yml /etc/accp/config.yml

FROM scratch

COPY --from=0 /go/bin/$APP_NAME /usr/bin/$APP_NAME
COPY --from=0 /go/bin/service /usr/bin/service
COPY --from=0 /etc/accp/config.yml /etc/accp/config.yml

USER 1000

# Set the entrypoint
ENTRYPOINT ["service"]