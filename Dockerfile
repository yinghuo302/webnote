FROM alpine:3.17 as builder
LABEL stage=go-builder
WORKDIR /app/
COPY ./ ./
RUN apk add --no-cache bash curl gcc git go musl-dev; \
	chmod +x build.sh; \
    bash build.sh

FROM alpine:3.17
LABEL MAINTAINER="zanilia"
VOLUME /opt/webnote/data
VOLUME /opt/webnote/img
WORKDIR /opt/webnote/
COPY --from=builder /app/bin/webnote ./
COPY run.sh /run.sh
RUN apk add --no-cache bash ca-certificates su-exec tzdata; \
    chmod +x /entrypoint.sh
EXPOSE 8000
CMD [ "/run.sh" ]
