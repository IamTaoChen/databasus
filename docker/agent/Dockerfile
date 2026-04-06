# ------------------------
# Downloader
# ------------------------
FROM alpine/curl AS downloader

ARG TARGETARCH
WORKDIR /tmp

RUN set -e; \
    echo "Building for arch: $TARGETARCH"; \
    curl -fL -o agent "https://app.databasus.com:443/api/v1/system/agent?arch=${TARGETARCH}"; \
    chmod +x agent;

# ------------------------
# Runtime
# ------------------------
FROM alpine/psql:latest

WORKDIR /opt/databasus
RUN apk add --no-cache ca-certificates tini

COPY --from=downloader /tmp/agent /opt/databasus/agent
RUN chmod +x /opt/databasus/agent

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENV DATABASUS_HOST=https://app.databasus.com:443
ENV DATABASUS_DB_ID=
ENV DATABASUS_TOKEN=
ENV PG_BACKUPER_USER=
ENV PG_BACKUPER_PASSWORD=
ENV PG_WAL_DIR=/opt/databasus/wal-queue
ENV PG_HOST=localhost
ENV PG_PORT=5432

VOLUME ["/opt/databasus/wal-queue"]

ENTRYPOINT ["tini", "--", "/entrypoint.sh"]