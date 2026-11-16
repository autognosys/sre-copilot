#!/usr/bin/env bash
# usage: bench/ask.sh <scenario> <model> "<question>"
set -euo pipefail
S=$1; M=$2; Q=$3
OUT=bench/${S}_${M}_$(date +%H%M%S).json
T0=$(date +%s)
curl -sS localhost:8080/api/chat -H 'Content-Type: application/json' -H 'Accept: application/json' \
  -d "$(jq -n --arg ask "$Q" --arg model "$M" '{ask:$ask,model:$model}')" > "$OUT" || { echo "curl failed - is the port-forward up?"; rm -f "$OUT"; exit 1; }
T1=$(date +%s)
[ -s "$OUT" ] || { echo "empty response - is the port-forward up?"; rm -f "$OUT"; exit 1; }
jq -r --arg t "$((T1-T0))" '.analysis, "---",
  "seconds: \($t)", "cost: \(.metadata.costs.total_cost)",
  "tool_calls: \(.tool_calls|length)",
  "prompt_tokens: \(.metadata.costs.prompt_tokens)",
  "completion_tokens: \(.metadata.costs.completion_tokens)"' "$OUT"
