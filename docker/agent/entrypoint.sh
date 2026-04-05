#!/usr/bin/env sh
set -eu

log_info() { echo "[INFO] $*"; }
log_error() { echo "[ERROR] $*" >&2; }
log_debug() { if [ "${AGENT_LOG_LEVEL:-info}" = "debug" ]; then echo "[DEBUG] $*"; fi; }

get_env() {
    printenv "$1" 2>/dev/null || true
}

load_secrets() {
    VAR_LIST="$1"
    VAR_PREFIX="${2:-}"

    [ -z "$VAR_LIST" ] && log_error "No variable list provided" && return 1

    for VAR in $(echo "$VAR_LIST" | tr ',' ' '); do
        log_debug "Processing variable: $VAR"
        FULL_VAR="${VAR_PREFIX}${VAR}"
        FILE_VAR="${FULL_VAR}_FILE"

        FILE_PATH=$(get_env "$FILE_VAR")
        if [ -n "$FILE_PATH" ] && [ -f "$FILE_PATH" ]; then
            VALUE=$(tr -d '\n' < "$FILE_PATH")
            log_info "Loading $FULL_VAR from file"
            export "$FULL_VAR=$VALUE"
        else
            log_debug "No file for $FULL_VAR, checking environment variable"
            VALUE=$(get_env "$FULL_VAR")
            if [ -n "$VALUE" ]; then
                log_info "Using $FULL_VAR (value hidden)"
            else
                log_error "$FULL_VAR is not set"
                exit 1
            fi
        fi
    done
}

clean_secrets() {
    VAR_LIST="$1"
    VAR_PREFIX="${2:-}"

    for VAR in $(echo "$VAR_LIST" | tr ',' ' '); do
        FULL_VAR="${VAR_PREFIX}${VAR}"
        FILE_VAR="${FULL_VAR}_FILE"
        FILE_PATH=$(get_env "$FILE_VAR")

        if [ -n "$FILE_PATH" ] && [ -f "$FILE_PATH" ]; then
            unset "$FULL_VAR"
            log_info "Unset $FULL_VAR"
        fi
    done
}

assert_env_not_empty() {
    VAR_NAME="$1"
    VAR_VALUE=$(get_env "$VAR_NAME")

    if [ -z "$VAR_VALUE" ]; then
        log_error "$VAR_NAME is empty"
        exit 1
    else
        log_info "$VAR_NAME is set (hidden)"
    fi
}

assert_envs_not_empty() {
    for VAR in $(echo "$1" | tr ',' ' '); do
        assert_env_not_empty "$VAR"
    done
}

# ------------------------
# Load secrets
# ------------------------

SECRETS_NAME="DATABASUS_TOKEN,PG_BACKUPER_PASSWORD"
load_secrets "$SECRETS_NAME"

REQUIRED_ENVS="DATABASUS_DB_ID,PG_BACKUPER_USER"
assert_envs_not_empty "$REQUIRED_ENVS"

# ------------------------
# Lifecycle
# ------------------------

stop_agent() {
    log_info "Stopping databasus agent..."
    /opt/databasus/agent stop || true
    exit 0
}

trap stop_agent TERM INT

log_info "Starting databasus agent..."

# sleep infinity

/opt/databasus/agent start \
    --databasus-host="${DATABASUS_HOST}" \
    --db-id="${DATABASUS_DB_ID}" \
    --token="${DATABASUS_TOKEN}" \
    --pg-host="${PG_HOST}" \
    --pg-port="${PG_PORT}" \
    --pg-user="${PG_BACKUPER_USER}" \
    --pg-password="${PG_BACKUPER_PASSWORD}" \
    --pg-wal-dir="${PG_WAL_DIR}" \
    --skip-update

sleep 1

# ------------------------
# Health monitoring
# ------------------------

log_info "Agent started, entering monitor loop"

while true; do
    if ! /opt/databasus/agent status >/dev/null 2>&1; then
        log_error "Agent is not running, exiting..."
        exit 1
    fi

    sleep 30 &
    wait $!
done