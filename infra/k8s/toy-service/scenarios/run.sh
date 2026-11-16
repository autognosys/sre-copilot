#!/usr/bin/env bash
# usage: run.sh <scenario> "<question>" [settle-seconds]
set -uo pipefail
T=infra/k8s/toy-service; S=$1; Q=$2; WAIT=${3:-60}
curl -sf -o /dev/null -H "Accept: application/json" localhost:8080/api/model || { echo "Holmes not reachable on :8080 - start the port-forward"; exit 1; }
restore() {
  kubectl -n default delete events --all >/dev/null 2>&1
  kubectl -n default delete deploy order-service --ignore-not-found --wait=true >/dev/null
  kubectl apply -f $T/deployment.yaml >/dev/null
  kubectl -n default patch svc order-service -p '{"spec":{"selector":{"app":"order-service"}}}' >/dev/null
  kubectl -n default rollout status deploy/order-service --timeout=120s >/dev/null
}
restore; trap restore EXIT
if [[ $S == svc-* ]]; then kubectl -n default patch svc order-service --patch-file $T/scenarios/$S.yaml
else kubectl -n default patch deploy order-service --patch-file $T/scenarios/$S.yaml; fi
sleep "$WAIT"
echo "=== ground truth: $S ==="
kubectl -n default get pods,svc,endpoints -o wide | tee bench/${S}_truth.txt
bench/ask.sh "$S" haiku  "$Q"
bench/ask.sh "$S" sonnet "$Q"
