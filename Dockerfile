FROM golang:1.23.3-alpine AS builder

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



FROM alpine:3.20

COPY --from=rootfs ["/", "/"]

WORKDIR /app

CMD [ "/app/tg-events-service" ]

LABEL maintainer="Aleksandar Puharic <aleksandar@puharic.com>" \
      org.opencontainers.image.documentation="https://github.com/N0rthernL1ghts/telegram-event-server/wiki" \
      org.opencontainers.image.source="https://github.com/N0rthernL1ghts/telegram-event-server" \
      org.opencontainers.image.description="Telegram Event Server 1.0.0 - Build ${TARGETPLATFORM}" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version="1.0.0"

ENV ALLOWED_ORIGINS="http://127.0.0.1"
EXPOSE 8080/TCP