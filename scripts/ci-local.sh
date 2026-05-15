#!/usr/bin/env bash
# Local mirror of .github/workflows/ci.yml — same stages and tools, no act.
# Run from anywhere: ./scripts/ci-local.sh
#
# Expects: Docker, Go (toolchain per pb_backend/go.mod), Node 20+, npm.
# Missing CLIs fall back to the same Docker images CI would pull indirectly.
#
# Semgrep is OFF by default locally: it downloads rule packs from semgrep.dev and
# often times out on flaky networks. GitHub Actions still runs Semgrep in ci.yml.
# To run Semgrep locally: CI_LOCAL_SEMGREP=1 ./scripts/ci-local.sh
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# Match container-security image tags (github.sha → local git commit or "local").
CI_SHA="$(git -C "$ROOT_DIR" rev-parse HEAD 2>/dev/null || echo local)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

job() {
  echo -e "\n${YELLOW}=== $1 ===${NC}"
}

ok() {
  echo -e "${GREEN}✓ $1${NC}"
}

die() {
  echo -e "${RED}error:${NC} $*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

ensure_golangci_lint() {
  local desired_ver="2.12.1"
  if command -v golangci-lint >/dev/null 2>&1; then
    if golangci-lint version 2>/dev/null | rg -q "version v?${desired_ver}"; then
      return 0
    fi
    echo -e "${YELLOW}info:${NC} replacing local golangci-lint with v${desired_ver}"
  fi
  local gopath_bin
  gopath_bin="$(go env GOPATH)/bin"
  mkdir -p "$gopath_bin"
  if [ -x "$gopath_bin/golangci-lint" ]; then
    if "$gopath_bin/golangci-lint" version 2>/dev/null | rg -q "version v?${desired_ver}"; then
      export PATH="$gopath_bin:$PATH"
      return 0
    fi
  fi
  echo -e "${YELLOW}info:${NC} installing golangci-lint v${desired_ver} locally via go install"
  # v2 module path is required for golangci-lint config version "2"
  GOBIN="$gopath_bin" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v${desired_ver}
  export PATH="$gopath_bin:$PATH"
  if ! command -v golangci-lint >/dev/null 2>&1; then
    die "failed to install golangci-lint"
  fi
  golangci-lint version | rg -q "version v?${desired_ver}" || die "unexpected golangci-lint version installed"
}

# CI uses actions/setup-go with pb_backend/go.mod; align with toolchain directive.
export GOTOOLCHAIN="${GOTOOLCHAIN:-local}"

run_ansible_lint() {
  local lintable=$1
  git config --global --add safe.directory "$ROOT_DIR" 2>/dev/null || true
  if command -v ansible-lint >/dev/null 2>&1; then
    ansible-lint "$lintable" -c "$ROOT_DIR/.ansible-lint"
  else
    docker run --rm --entrypoint "" -v "$ROOT_DIR:/w" -w /w cytopia/ansible-lint:latest \
      sh -c "git config --global --add safe.directory /w && ansible-lint $lintable -c .ansible-lint"
  fi
}

warn_env() {
  if command -v node >/dev/null 2>&1; then
    case "$(node -p "process.version.slice(1).split('.')[0]" 2>/dev/null || echo 0)" in
      20) ;;
      *) echo -e "${YELLOW}warning:${NC} GitHub CI uses Node 20; you have $(node -v 2>/dev/null || echo unknown)" >&2 ;;
    esac
  fi
}

require_cmd docker
require_cmd go
require_cmd npm
warn_env

# --- secrets-scan (gitleaks) ---
job "secrets-scan"
if command -v gitleaks >/dev/null 2>&1; then
  gitleaks detect --source . --config .gitleaks.toml --redact
else
  docker run --rm -v "$ROOT_DIR:/repo" -w /repo zricethezav/gitleaks:latest detect \
    --source . --config .gitleaks.toml --redact
fi
ok "secrets-scan"

# --- ansible-lint (deploy/) ---
job "ansible-lint"
run_ansible_lint deploy/
ok "ansible-lint"

# --- benchmark-tooling-lint (benchmark/ Ansible) ---
job "benchmark-tooling-lint"
run_ansible_lint benchmark/
ok "benchmark-tooling-lint"

# --- backend-test (unit tests) ---
job "backend-test"
(
  cd pb_backend
  go test ./...
)
ok "backend-test"

# --- buf-lint (Phase 2 gRPC proto) ---
job "buf-lint"
ensure_buf() {
  local desired_ver="1.50.0"
  if command -v buf >/dev/null 2>&1; then
    if buf --version 2>/dev/null | rg -q "^${desired_ver}$"; then
      return 0
    fi
  fi
  local gopath_bin
  gopath_bin="$(go env GOPATH)/bin"
  if [ ! -x "$gopath_bin/buf" ]; then
    GOBIN="$gopath_bin" go install github.com/bufbuild/buf/cmd/buf@v${desired_ver}
  fi
  export PATH="$gopath_bin:$PATH"
}
ensure_buf
(
  cd pb_backend
  buf lint
  buf format -d --exit-code
)
ok "buf-lint"

# --- backend-lint ---
job "backend-lint"
ensure_golangci_lint
(
  cd pb_backend
  go vet ./...
  test -z "$(gofmt -l .)"
  golangci-lint run
)
ok "backend-lint"

# --- backend-sast (gosec, govulncheck; semgrep only if CI_LOCAL_SEMGREP=1) ---
job "backend-sast"
(
  cd pb_backend
  if command -v gosec >/dev/null 2>&1; then
    gosec -exclude=G104,G114,G115 ./...
  else
    docker run --rm -v "$ROOT_DIR:/src" -w /src/pb_backend securego/gosec:latest \
      -exclude=G104,G114,G115 ./...
  fi

  GOVULNCHECK_BIN="$(go env GOPATH)/bin/govulncheck"
  if [ ! -x "$GOVULNCHECK_BIN" ]; then
    go install golang.org/x/vuln/cmd/govulncheck@latest
    GOVULNCHECK_BIN="$(go env GOPATH)/bin/govulncheck"
  fi
  "$GOVULNCHECK_BIN" ./...
)
if [ "${CI_LOCAL_SEMGREP:-0}" = "1" ]; then
  if command -v semgrep >/dev/null 2>&1; then
    semgrep --config p/ci --config p/golang --config p/owasp-top-ten --config p/dockerfile .
  else
    docker run --rm -v "$ROOT_DIR:/src" semgrep/semgrep:latest semgrep \
      --config p/ci --config p/golang --config p/owasp-top-ten --config p/dockerfile /src
  fi
else
  echo -e "${YELLOW}info:${NC} skipping Semgrep (set CI_LOCAL_SEMGREP=1 to run; CI still runs it on push)"
fi
ok "backend-sast"

# --- frontend-lint (npm ci + eslint only; mirrors frontend-lint job) ---
job "frontend-lint"
(
  cd pb_frontend
  npm ci
  npm run lint
)
ok "frontend-lint"

# --- frontend-security (npm ci + audit; semgrep only if CI_LOCAL_SEMGREP=1) ---
job "frontend-security"
(
  cd pb_frontend
  npm ci
  # Advisory bulk fetch can flake with "socket hang up"; extra retries stay inside one audit run.
  NPM_CONFIG_FETCH_RETRIES=8 NPM_CONFIG_FETCH_RETRY_MINTIMEOUT=20000 \
    npm audit --audit-level=high
)
if [ "${CI_LOCAL_SEMGREP:-0}" = "1" ]; then
  if command -v semgrep >/dev/null 2>&1; then
    semgrep --config p/javascript --config p/owasp-top-ten .
  else
    docker run --rm -v "$ROOT_DIR:/src" semgrep/semgrep:latest semgrep \
      --config p/javascript --config p/owasp-top-ten /src
  fi
else
  echo -e "${YELLOW}info:${NC} skipping Semgrep (set CI_LOCAL_SEMGREP=1 to run; CI still runs it on push)"
fi
ok "frontend-security"

# --- dockerfile-lint (hadolint + .hadolint.yaml) ---
job "dockerfile-lint"
HADOLINT=(hadolint -c "$ROOT_DIR/.hadolint.yaml")
if command -v hadolint >/dev/null 2>&1; then
  "${HADOLINT[@]}" "$ROOT_DIR/pb_backend/Dockerfile"
  "${HADOLINT[@]}" "$ROOT_DIR/pb_frontend/Dockerfile"
else
  docker run --rm -v "$ROOT_DIR:/w" -w /w hadolint/hadolint \
    hadolint -c .hadolint.yaml pb_backend/Dockerfile
  docker run --rm -v "$ROOT_DIR:/w" -w /w hadolint/hadolint \
    hadolint -c .hadolint.yaml pb_frontend/Dockerfile
fi
ok "dockerfile-lint"

# --- docker-build (build images; mirrors docker-build job) ---
job "docker-build"
docker build -t "pb-backend:${CI_SHA}" pb_backend/
docker build -t "pb-frontend:${CI_SHA}" pb_frontend/
ok "docker-build"

echo -e "\n${GREEN}Local CI completed (parity with .github/workflows/ci.yml: ansible-lint, benchmark-tooling-lint, backend-test, backend-lint, …).${NC}"
