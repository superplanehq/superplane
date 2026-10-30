#!/bin/bash

set -euo pipefail

matches=$(
	grep -R --include='*.go' --exclude='*_test.go' -n 'google\.golang\.org/grpc/status' pkg/grpc/actions ee/billing/organizations ee/billing/http 2>/dev/null || true
)

if [[ -z "$matches" ]]; then
	exit 0
fi

echo "$matches" >&2
echo >&2
echo "Core and EE handlers must not import google.golang.org/grpc/status." >&2
echo "Use github.com/superplanehq/superplane/pkg/grpc/errors instead so errors reach the grpc-gateway sanitizer." >&2
exit 1
