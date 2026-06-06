#!/bin/bash
# Tek satır deploy + verify
set +e
echo "=== STEP 1: Deploy ==="
curl -sL "https://api.github.com/repos/isaakkaya44-afk/rmm/contents/deploy-vps.sh" -H "Accept: application/vnd.github.v3.raw" -o /tmp/deploy.sh
ls -lh /tmp/deploy.sh
head -3 /tmp/deploy.sh
echo ""
echo "=== STEP 2: Run deploy ==="
bash /tmp/deploy.sh 2>&1 | tail -80
echo ""
echo "=== STEP 3: Container status ==="
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
echo ""
echo "=== STEP 4: Login test ==="
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login -H "Content-Type: application/json" -d '{"email":"admin@rmm.local","password":"admin123"}' | python3 -c "import sys,json;print(json.load(sys.stdin).get('access_token',''))" 2>/dev/null)
echo "Token: ${TOKEN:0:30}..."
echo ""
echo "=== STEP 5: Devices list ==="
curl -s http://localhost:8080/api/v1/devices
echo ""
echo "=== STEP 6: Test heartbeat ==="
curl -s -X POST http://localhost:8080/api/v1/devices/heartbeat -H "Content-Type: application/json" -d '{"hostname":"VPS-LOCAL-TEST","os_version":"Ubuntu 26.04","cpu_percent":15.5,"ram_percent":45.0,"ram_used_mb":3600,"ram_total_mb":8000,"disk_percent":35.0,"disk_used_mb":52000,"disk_total_mb":150000,"uptime_seconds":86400,"cpu_model":"Hetzner CX","cpu_cores":4,"pos_running":false,"mssql_running":false,"agent_version":"1.0.0"}'
echo ""
echo "=== STEP 7: DB check ==="
docker exec rmm-postgres-1 psql -U rmm -d rmm_platform -c "SELECT hostname, is_online, last_heartbeat, mssql_status, pos_process_status FROM devices ORDER BY last_heartbeat DESC NULLS LAST LIMIT 5;"
echo ""
echo "=== STEP 8: API logs (last 20) ==="
docker logs rmm-api-1 --tail 20 2>&1
echo "=== DONE ==="
