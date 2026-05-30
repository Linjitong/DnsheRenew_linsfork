#!/usr/bin/env bash

set -uo pipefail

action_dir="${GITHUB_ACTION_PATH:?GITHUB_ACTION_PATH is required}"
cd "$action_dir"

stdout_file="$(mktemp)"
stderr_file="$(mktemp)"

cleanup() {
  rm -f "$stdout_file" "$stderr_file"
}
trap cleanup EXIT

export DNSHE_API_KEYS="${INPUT_API_KEYS:-}"
export DNSHE_API_SECRETS="${INPUT_API_SECRETS:-}"
export DNSHE_API_BASE_URL="${INPUT_API_BASE_URL:-}"
export DNSHE_DRY_RUN="${INPUT_DRY_RUN:-}"
export DNSHE_RUN_TIMEOUT="${INPUT_RUN_TIMEOUT:-${DNSHE_RUN_TIMEOUT:-}}"
export DNSHE_DEBUG="${INPUT_DEBUG:-}"
export DNSHE_NOTIFY_TELEGRAM_BOT_TOKEN="${INPUT_TELEGRAM_BOT_TOKEN:-}"
export DNSHE_NOTIFY_TELEGRAM_CHAT_ID="${INPUT_TELEGRAM_CHAT_ID:-}"
export DNSHE_NOTIFY_TELEGRAM_MESSAGE_THREAD_ID="${INPUT_TELEGRAM_MESSAGE_THREAD_ID:-}"
export DNSHE_NOTIFY_TELEGRAM_PARSE_MODE="${INPUT_TELEGRAM_PARSE_MODE:-}"
export DNSHE_NOTIFY_LARK_WEBHOOK_URL="${INPUT_LARK_WEBHOOK_URL:-}"
export DNSHE_NOTIFY_LARK_APP_ID="${INPUT_LARK_APP_ID:-}"
export DNSHE_NOTIFY_LARK_APP_SECRET="${INPUT_LARK_APP_SECRET:-}"
export DNSHE_NOTIFY_LARK_RECEIVER_TYPE="${INPUT_LARK_RECEIVER_TYPE:-}"
export DNSHE_NOTIFY_LARK_RECEIVERS="${INPUT_LARK_RECEIVERS:-}"
export DNSHE_NOTIFY_WEBHOOK_URL="${INPUT_WEBHOOK_URL:-}"
export DNSHE_NOTIFY_WEBHOOK_TOKEN="${INPUT_WEBHOOK_TOKEN:-}"

set +e
go run ./cmd/dnsherene >"$stdout_file" 2>"$stderr_file"
exit_code=$?
set -e

renewed_total="$(awk -F= '/^renewed_total=/{value=$2} END{print value}' "$stdout_file")"
notification_error_count="$(awk -F= '/^notification_error_count=/{value=$2} END{print value}' "$stderr_file")"
error_count="$(awk -F= '/^error_count=/{value=$2} END{print value}' "$stderr_file")"

if [[ -z "$renewed_total" ]]; then
  renewed_total="0"
fi
if [[ -z "$notification_error_count" ]]; then
  notification_error_count="0"
fi
if [[ -z "$error_count" ]]; then
  error_count="0"
fi

success="false"
if [[ "$exit_code" -eq 0 ]]; then
  success="true"
fi

if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  {
    echo "renewed-total=$renewed_total"
    echo "notification-error-count=$notification_error_count"
    echo "error-count=$error_count"
    echo "success=$success"
  } >>"$GITHUB_OUTPUT"
fi

if [[ -s "$stdout_file" ]]; then
  cat "$stdout_file"
fi
if [[ -s "$stderr_file" ]]; then
  cat "$stderr_file" >&2
fi

if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  {
    echo "## DNSHE Renew"
    echo
    echo "- success: $success"
    echo "- renewed-total: $renewed_total"
    echo "- notification-error-count: $notification_error_count"
    echo "- error-count: $error_count"
  } >>"$GITHUB_STEP_SUMMARY"
fi

exit "$exit_code"
