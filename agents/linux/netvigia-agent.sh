#!/usr/bin/env bash
# Métricas básicas do host Linux e número de containers Docker em execução.
set -euo pipefail
SERVER_URL="${NETVIGIA_URL:-http://127.0.0.1:8787}"
DEVICE_ID="${NETVIGIA_DEVICE_ID:?Defina NETVIGIA_DEVICE_ID}"
API_KEY="${NETVIGIA_API_KEY:?Defina NETVIGIA_API_KEY}"
read -r _ u n sy id iw irq sirq st _ < /proc/stat
idle1=$((id+iw)); total1=$((u+n+sy+id+iw+irq+sirq+st)); sleep 1
read -r _ u n sy id iw irq sirq st _ < /proc/stat
idle2=$((id+iw)); total2=$((u+n+sy+id+iw+irq+sirq+st)); dt=$((total2-total1)); di=$((idle2-idle1))
cpu=$(awk -v d="$di" -v t="$dt" 'BEGIN{if(t>0)printf "%.2f",100*(1-d/t);else print 0}')
read -r total available < <(awk '/MemTotal:/{t=$2}/MemAvailable:/{a=$2}END{print t,a}' /proc/meminfo)
mem=$(awk -v t="$total" -v a="$available" 'BEGIN{if(t>0)printf "%.2f",100*(1-a/t);else print 0}')
containers=0; if command -v docker >/dev/null 2>&1; then containers=$(docker ps -q 2>/dev/null | wc -l | tr -d ' '); fi
payload=$(printf '{"device_id":"%s","name":"%s","metrics":{"cpu_percent":%s,"memory_total_kb":%s,"memory_used_percent":%s,"docker_running_containers":%s}}' "$DEVICE_ID" "$(hostname)" "$cpu" "$total" "$mem" "$containers")
curl -fsS -X POST "$SERVER_URL/api/agent/metrics" -H "X-API-Key: $API_KEY" -H 'Content-Type: application/json' --data "$payload"
