#!/usr/bin/env bash
# Quick smoke test against a running instance of the API.
# Usage: ./scripts/smoke_test.sh [base_url]
set -euo pipefail

BASE_URL="${1:-http://localhost:8080}"
SAMPLE_FILE="$(dirname "$0")/../sample_data/Sample_Employee_data.xlsx"

echo "==> Health check"
curl -sf "$BASE_URL/health" | jq .

echo "==> Uploading sample file"
UPLOAD_RESP=$(curl -sf -X POST "$BASE_URL/api/v1/upload" -F "file=@${SAMPLE_FILE}")
echo "$UPLOAD_RESP" | jq .
JOB_ID=$(echo "$UPLOAD_RESP" | jq -r .data.job_id)

echo "==> Polling job status (job_id=$JOB_ID)"
for i in $(seq 1 15); do
  STATUS_RESP=$(curl -sf "$BASE_URL/api/v1/upload/status/$JOB_ID")
  STATUS=$(echo "$STATUS_RESP" | jq -r .data.status)
  echo "  attempt $i: status=$STATUS"
  if [[ "$STATUS" == "completed" || "$STATUS" == "failed" ]]; then
    echo "$STATUS_RESP" | jq .
    break
  fi
  sleep 2
done

echo "==> Listing first page of employees"
curl -sf "$BASE_URL/api/v1/employees?page=1&page_size=5" | jq .

echo "==> Fetching employee #1"
curl -sf "$BASE_URL/api/v1/employees/1" | jq .

echo "==> Updating employee #1"
curl -sf -X PUT "$BASE_URL/api/v1/employees/1" \
  -H "Content-Type: application/json" \
  -d '{"city": "Updated City"}' | jq .

echo "Smoke test complete."
