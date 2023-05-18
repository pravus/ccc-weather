FROM alpine:3 AS builder

RUN apk --no-cache update \
 && apk --no-cache upgrade \
 && apk --no-cache add ca-certificates go

WORKDIR /usr/src/ccc-weather

COPY go.mod ./
COPY go.sum ./
COPY main.go ./

RUN go test -race ./... \
 && CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' -o ccc-weather


FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/src/ccc-weather/ccc-weather /ccc-weather

ENTRYPOINT ["/ccc-weather"]
