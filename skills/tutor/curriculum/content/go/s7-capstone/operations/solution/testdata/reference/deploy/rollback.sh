#!/usr/bin/env bash
# Roll back to the previous revision. Rehearsed, not improvised: this is the
# script you run before you understand the incident, not after.
#
#   ./deploy/rollback.sh
set -euo pipefail

namespace="${NAMESPACE:-tinysvc}"

echo "==> rolling back deployment/tinysvc in ${namespace}"
kubectl -n "${namespace}" rollout undo deployment/tinysvc
kubectl -n "${namespace}" rollout status deployment/tinysvc --timeout=120s

echo "==> now serving:"
kubectl -n "${namespace}" get deployment tinysvc \
	-o jsonpath='{.spec.template.spec.containers[0].image}{"\n"}'
