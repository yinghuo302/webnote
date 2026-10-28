FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /webnote .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && addgroup -S web && adduser -S -G web web && mkdir -p /opt/webnote/data && chown -R web:web /opt/webnote/data
COPY --from=build /webnote /opt/webnote/bin/webnote
USER web
ENV WEBNOTE_ADDRESS=0.0.0.0 WEBNOTE_DATA_DIR=/opt/webnote/data/legacy WEBNOTE_IMG_DIR=/opt/webnote/data/img
WORKDIR /opt/webnote
VOLUME ["/opt/webnote/data"]
EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=3s CMD wget -q -O /dev/null http://127.0.0.1:8000/api/ping || exit 1
ENTRYPOINT ["/opt/webnote/bin/webnote"]
