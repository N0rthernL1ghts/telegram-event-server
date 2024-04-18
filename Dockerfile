FROM golang:1.22.2-alpine AS builder

WORKDIR /app

RUN set -eux \
    && apk add --no-cache --update make


COPY ["./go.mod", "./go.sum", "/app/"]
RUN set -eux \
    && go mod download

COPY ["./server/", "/app/server/"]
COPY ["./Makefile", "/app/"]

RUN set -eux \
    && export CGO_ENABLED=0 \
    && make build



FROM scratch AS rootfs

COPY --from=builder --chmod=0775 ["/app/build/release/tg-events-service", "/app/tg-events-service"]



FROM alpine:3.19

COPY --from=rootfs ["/", "/"]

WORKDIR /app

CMD [ "/app/tg-events-service" ]



ENV ALLOWED_ORIGINS="http://127.0.0.1"
EXPOSE 8080/TCP