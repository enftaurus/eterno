#!/usr/bin/env bash

# ==============================================================================
# Eterno — Development Environment Orchestrator
# Starts PostgreSQL (Docker), installs dependencies, runs migrations & seed,
# and starts the Backend API, Background Worker, and React Frontend.
# ==============================================================================

set -eo pipefail

# ANSI Color codes for clean output
BOLD='\033[1m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_ROOT"

# Ensure Go and Node are in PATH if installed in common paths
if ! command -v go &>/dev/null && [ -x "/usr/local/go/bin/go" ]; then
    export PATH="/usr/local/go/bin:$PATH"
fi

if ! command -v node &>/dev/null && [ -d "$HOME/.nvm" ]; then
    export NVM_DIR="$HOME/.nvm"
    [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
fi

echo -e "${BOLD}${CYAN}"
echo "  ███████╗████████╗███████╗██████╗ ███╗   ██╗ ██████╗ "
echo "  ██╔════╝╚══██╔══╝██╔════╝██╔══██╗████╗  ██║██╔═══██╗"
echo "  █████╗     ██║   █████╗  ██████╔╝██╔██╗ ██║██║   ██║"
echo "  ██╔══╝     ██║   ██╔══╝  ██╔══██╗██║╚██╗██║██║   ██║"
echo "  ███████╗   ██║   ███████╗██║  ██║██║ ╚████║╚██████╔╝"
echo "  ╚══════╝   ╚═╝   ╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝ ╚═════╝ "
echo -e "${NC}"
echo -e "${BOLD}Developer Intelligence & College Coding Challenges${NC}"
echo -e "Starting all services in development mode...\n"

# Verify Prerequisites
echo -e "${BLUE}[1/5] Checking environment prerequisites...${NC}"

if ! command -v docker &>/dev/null; then
    echo -e "${RED}Error: Docker is not installed or not in PATH.${NC}"
    exit 1
fi

if ! command -v go &>/dev/null; then
    echo -e "${RED}Error: Go compiler is not installed or not in PATH.${NC}"
    exit 1
fi

if ! command -v node &>/dev/null || ! command -v npm &>/dev/null; then
    echo -e "${RED}Error: Node.js and npm are required.${NC}"
    exit 1
fi

echo -e "  ✔ Docker:  $(docker --version)"
echo -e "  ✔ Go:      $(go version | awk '{print $3}')"
echo -e "  ✔ Node.js: $(node --version)"
echo -e "  ✔ npm:     v$(npm --version)"

# 1. PostgreSQL in Docker
echo -e "\n${BLUE}[2/5] Initializing PostgreSQL container (eterno-postgres)...${NC}"
DB_CONTAINER="eterno-postgres"
DB_PORT="5434"
DB_USER="eterno"
DB_PASSWORD="eterno_secret"
DB_NAME="eterno"

if [ "$(docker ps -q -f name=^/${DB_CONTAINER}$)" ]; then
    echo -e "  ✔ Database container '${DB_CONTAINER}' is already running on port ${DB_PORT}."
elif [ "$(docker ps -aq -f status=exited -f name=^/${DB_CONTAINER}$)" ]; then
    echo -e "  Starting existing stopped container '${DB_CONTAINER}'..."
    docker start "$DB_CONTAINER" >/dev/null
else
    echo -e "  Creating and starting new container '${DB_CONTAINER}' on port ${DB_PORT}..."
    docker run -d \
        --name "$DB_CONTAINER" \
        -e POSTGRES_DB="$DB_NAME" \
        -e POSTGRES_USER="$DB_USER" \
        -e POSTGRES_PASSWORD="$DB_PASSWORD" \
        -p 127.0.0.1:"$DB_PORT":5432 \
        postgres:16-alpine >/dev/null
fi

echo -e "  Waiting for PostgreSQL to accept connections..."
MAX_RETRIES=30
RETRY_COUNT=0
until docker exec "$DB_CONTAINER" pg_isready -U "$DB_USER" -d "$DB_NAME" &>/dev/null || [ $RETRY_COUNT -eq $MAX_RETRIES ]; do
    sleep 1
    RETRY_COUNT=$((RETRY_COUNT + 1))
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    echo -e "${RED}Error: Timed out waiting for PostgreSQL to become healthy.${NC}"
    exit 1
fi
echo -e "  ✔ PostgreSQL is ready."

# 2. Database Migrations and Demo Seed
echo -e "\n${BLUE}[3/5] Verifying database schema and seed data...${NC}"
HAS_TABLES=$(docker exec "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -tAc "SELECT to_regclass('institutions');" 2>/dev/null || true)

if [ "$HAS_TABLES" != "institutions" ]; then
    echo -e "  Applying initial database schema migration (000001_initial_schema.up.sql)..."
    docker exec -i "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" < "$PROJECT_ROOT/backend/migrations/000001_initial_schema.up.sql" >/dev/null
    echo -e "  ✔ Schema created."
fi

USER_COUNT=$(docker exec "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -tAc "SELECT count(*) FROM users;" 2>/dev/null || echo "0")
if [ "$USER_COUNT" -eq "0" ]; then
    echo -e "  Applying demonstration seed data for Stanford School of Engineering (seed.sql)..."
    docker exec -i "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" < "$PROJECT_ROOT/backend/migrations/seed.sql" >/dev/null
    echo -e "  ✔ Demo dataset applied."
else
    echo -e "  ✔ Database schema and seed data verified (${USER_COUNT} users registered)."
fi

# 3. Dependencies & Build
echo -e "\n${BLUE}[4/5] Installing dependencies and building binaries...${NC}"

echo -e "  Backend: Downloading Go modules and building binaries..."
cd "$PROJECT_ROOT/backend"
go mod download
mkdir -p bin
go build -o bin/api ./cmd/api
go build -o bin/worker ./cmd/worker
echo -e "  ✔ Backend binaries built (bin/api, bin/worker)."

echo -e "  Frontend: Installing npm packages..."
cd "$PROJECT_ROOT/frontend"
npm install --prefer-offline --no-audit --no-fund >/dev/null 2>&1
echo -e "  ✔ Frontend dependencies installed."

# 4. Service Startup & Process Management
echo -e "\n${BLUE}[5/5] Launching all services...${NC}"

# Check for stale processes on port 8080 and port 5173
if lsof -ti:8080 &>/dev/null; then
    echo -e "${YELLOW}  Freeing port 8080 from previous process...${NC}"
    kill -9 $(lsof -ti:8080) 2>/dev/null || true
fi
if lsof -ti:5173 &>/dev/null; then
    echo -e "${YELLOW}  Freeing port 5173 from previous process...${NC}"
    kill -9 $(lsof -ti:5173) 2>/dev/null || true
fi

# Process cleanup handler
PIDS=()
cleanup() {
    echo -e "\n${YELLOW}Shutting down Eterno services gracefully...${NC}"
    for pid in "${PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
        fi
    done
    wait 2>/dev/null || true
    echo -e "${GREEN}All Eterno development services stopped. Goodbye!${NC}"
    exit 0
}
trap cleanup SIGINT SIGTERM EXIT

# Start API Server
export PORT="8080"
export ENV="development"
export DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@127.0.0.1:${DB_PORT}/${DB_NAME}?sslmode=disable"
export JWT_SECRET="eterno-super-secret-jwt-key-2026-production-ready"
export CORS_ALLOWED_ORIGINS="http://localhost:5173,http://localhost:3000,http://127.0.0.1:5173"
export MOCK_EXTERNAL_PROVIDERS="true"

"$PROJECT_ROOT/backend/bin/api" > "$PROJECT_ROOT/backend.log" 2>&1 &
API_PID=$!
PIDS+=("$API_PID")
echo -e "  ✔ Backend API Server started on port 8080 (PID: ${API_PID})"

# Wait for API server health
until curl -s http://127.0.0.1:8080/healthz | grep -q "healthy"; do
    if ! kill -0 "$API_PID" 2>/dev/null; then
        echo -e "${RED}Error: Backend API server exited unexpectedly. Check backend.log:${NC}"
        tail -n 20 "$PROJECT_ROOT/backend.log"
        exit 1
    fi
    sleep 0.5
done

# Start Background Worker
"$PROJECT_ROOT/backend/bin/worker" > "$PROJECT_ROOT/worker.log" 2>&1 &
WORKER_PID=$!
PIDS+=("$WORKER_PID")
echo -e "  ✔ Background Verification Worker started (PID: ${WORKER_PID})"

# Start Vite Frontend Dev Server
cd "$PROJECT_ROOT/frontend"
npm run dev -- --host 127.0.0.1 --port 5173 > "$PROJECT_ROOT/frontend.log" 2>&1 &
FRONTEND_PID=$!
PIDS+=("$FRONTEND_PID")
echo -e "  ✔ React Frontend Dev Server started on port 5173 (PID: ${FRONTEND_PID})"

# Wait briefly for Vite to spin up
sleep 1

echo -e "\n${BOLD}${GREEN}================================================================${NC}"
echo -e "${BOLD}${GREEN}  ✨ ETERNO DEVELOPMENT ENVIRONMENT IS READY! ✨${NC}"
echo -e "${BOLD}${GREEN}================================================================${NC}"
echo -e "  🌐 ${BOLD}Frontend Portal:${NC}    ${CYAN}http://localhost:5173${NC}"
echo -e "  ⚡ ${BOLD}Backend REST API:${NC}   ${CYAN}http://127.0.0.1:8080/api/v1${NC}"
echo -e "  🩺 ${BOLD}Health Endpoint:${NC}    ${CYAN}http://127.0.0.1:8080/healthz${NC}"
echo -e "  🐘 ${BOLD}PostgreSQL DB:${NC}      ${CYAN}127.0.0.1:5434 (${DB_NAME})${NC}"
echo -e "  📄 ${BOLD}Logs Available:${NC}     backend.log, worker.log, frontend.log"
echo -e ""
echo -e "  ${BOLD}Demo Credentials (Stanford Engineering):${NC}"
echo -e "  👤 ${BOLD}College Admin:${NC}      admin@stanford.edu  / Password123!"
echo -e "  👤 ${BOLD}Student (Alice):${NC}    alice@stanford.edu  / Password123!"
echo -e "  👤 ${BOLD}Student (Bob):${NC}      bob@stanford.edu    / Password123!"
echo -e "${BOLD}${GREEN}================================================================${NC}"
echo -e "${YELLOW}Press [Ctrl+C] at any time to gracefully stop all services.${NC}\n"

# Keep script running to monitor child processes and stream logs if desired
wait
