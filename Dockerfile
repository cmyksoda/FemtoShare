FROM golang:1.26.1 AS builder

ARG TARGETPLATFORM

# Copy the whole checkout so Go's VCS stamp only reports a dirty tree when the
# working tree really is dirty.
COPY . /app

WORKDIR /app

RUN TARGETPLATFORM="${TARGETPLATFORM}" \
    ./dev-scripts/build-backend "prod"

FROM scratch AS artifact
COPY --from=builder /app/bin/femtoshare ./

FROM debian:stable-20240311-slim AS litestream_downloader

ARG TARGETPLATFORM
ARG litestream_version="v0.3.13"

WORKDIR /litestream

RUN set -x && \
    apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
      ca-certificates \
      wget

RUN set -x && \
    if [ "$TARGETPLATFORM" = "linux/arm/v7" ]; then \
      ARCH="arm7" ; \
    elif [ "$TARGETPLATFORM" = "linux/arm64" ]; then \
      ARCH="arm64" ; \
    else \
      ARCH="amd64" ; \
    fi && \
    set -u && \
    litestream_binary_tgz_filename="litestream-${litestream_version}-linux-${ARCH}.tar.gz" && \
    wget "https://github.com/benbjohnson/litestream/releases/download/${litestream_version}/${litestream_binary_tgz_filename}" && \
    mv "${litestream_binary_tgz_filename}" litestream.tgz
RUN tar -xvzf litestream.tgz

FROM alpine:3.22

RUN apk add --no-cache bash ffmpeg

COPY --from=builder /app/bin/femtoshare /app/femtoshare
COPY --from=litestream_downloader /litestream/litestream /app/litestream
COPY ./docker-entrypoint /app/docker-entrypoint
COPY ./litestream.yml /etc/litestream.yml
COPY ./LICENSE /app/LICENSE

ENV LITESTREAM_RETENTION=72h
ENV LITESTREAM_PATH=db

WORKDIR /app

ENTRYPOINT ["/app/docker-entrypoint"]
CMD ["-db", "/data/store.db"]
