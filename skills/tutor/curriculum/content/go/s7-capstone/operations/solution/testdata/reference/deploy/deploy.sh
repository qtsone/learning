#!/usr/bin/env bash
# Deploy one immutable version of tinysvc. Every step is here rather than in
# somebody's shell history, so the deploy is repeatable and reviewable.
#
#   ./deploy/deploy.sh v1.4.2
set -euo pipefail

version="${1:?usage: deploy.sh <version tag>}"
registry="${REGISTRY:-registry.example.internal/tinysvc}"
namespace="${NAMESPACE:-tinysvc}"
image="${registry}:${version}"

echo "==> building ${image}"
docker build -t "${image}" .

echo "==> pushing ${image}"
docker push "${image}"

echo "==> rolling out to ${namespace}"
kubectl -n "${namespace}" set image deployment/tinysvc tinysvc="${image}"
kubectl -n "${namespace}" rollout status deployment/tinysvc --timeout=120s

echo "==> deployed ${image}"
echo "    roll back with: ./deploy/rollback.sh"
