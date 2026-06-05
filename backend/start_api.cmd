@echo off
SET DB_HOST=localhost
SET DB_PORT=5432
SET DB_USER=rmm
SET DB_PASSWORD=rmm_password
SET DB_NAME=rmm_platform
SET JWT_SECRET=rmm-platform-secret-key-change-in-production
SET SERVER_PORT=8080
SET GIN_MODE=release
SET PATH=C:\Users\JWPOS\Documents\go\go\bin;%PATH%
"C:\Users\JWPOS\Documents\rmm-platform\backend\rmm-api.exe"
