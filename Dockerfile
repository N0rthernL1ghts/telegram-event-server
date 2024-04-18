FROM golang:1.22.2-alpine AS builder

WORKDIR /app

COPY ["./server/", "/app/server/"]
COPY ["./go.mod", "./go.sum", "./Makefile", "/app/"]

RUN set -eux \
    && apk add --no-cache make \
    && make build



FROM scratch AS rootfs

COPY --from=builder --chmod=0775 ["/app/build/release/tg-events-service", "/app/tg-events-service"]



FROM alpine:3.19

COPY --from=rootfs ["/", "/"]

WORKDIR /app

CMD [ "/app/tg-events-service" ]

EXPOSE 8080/TCP