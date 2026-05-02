#!/usr/bin/env bash
# Local mirror of .github/workflows/ci.yml — same stages and tools, no act.
# Run from anywhere: ./scripts/ci-local.sh
#
# Expects: Docker, Go (1.23.x / toolchain per pb_backend/go.mod), Node 20+, npm.
# Missing CLIs fall back to the same Docker images CI would pull indirectly.
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

# CI uses actions/setup-go with go 1.23.6; align toolchain for vet/lint/SAST.
export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.23.6}"

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

# --- backend-lint ---
job "backend-lint"
(
  cd pb_backend
  go vet ./...
  test -z "$(gofmt -l .)"
  if command -v golangci-lint >/dev/null 2>&1; then
    golangci-lint run
  else
    docker run --rm -v "$ROOT_DIR:/src" -w /src/pb_backend \
      golangci/golangci-lint:v1.62.0 golangci-lint run
  fi
)
ok "backend-lint"

# --- backend-sast (gosec, govulncheck, semgrep: p/ci golang owasp dockerfile) ---
job "backend-sast"
(
  cd pb_backend
  if command -v gosec >/dev/null 2>&1; then
    gosec ./...
  else
    docker run --rm -v "$ROOT_DIR:/src" -w /src/pb_backend securego/gosec:latest ./...
  fi

  GOVULNCHECK_BIN="$(go env GOPATH)/bin/govulncheck"
  if [ ! -x "$GOVULNCHECK_BIN" ]; then
    go install golang.org/x/vuln/cmd/govulncheck@latest
    GOVULNCHECK_BIN="$(go env GOPATH)/bin/govulncheck"
  fi
  "$GOVULNCHECK_BIN" ./...
)
if command -v semgrep >/dev/null 2>&1; then
  semgrep --config p/ci --config p/golang --config p/owasp-top-ten --config p/dockerfile .
else
  docker run --rm -v "$ROOT_DIR:/src" semgrep/semgrep:latest semgrep \
    --config p/ci --config p/golang --config p/owasp-top-ten --config p/dockerfile /src
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

# --- frontend-security (npm ci + audit + semgrep js; mirrors frontend-security job) ---
job "frontend-security"
(
  cd pb_frontend
  npm ci
  npm audit --audit-level=high
)
if command -v semgrep >/dev/null 2>&1; then
  semgrep --config p/javascript --config p/owasp-top-ten .
else
  docker run --rm -v "$ROOT_DIR:/src" semgrep/semgrep:latest semgrep \
    --config p/javascript --config p/owasp-top-ten /src
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

# --- iac-scan (trivy config, CRITICAL+HIGH, exit 1) ---
job "iac-scan"
TRIVY_CONFIG_FLAGS=(config --severity CRITICAL,HIGH --exit-code 1 .)
if command -v trivy >/dev/null 2>&1; then
  trivy "${TRIVY_CONFIG_FLAGS[@]}"
else
  docker run --rm -v "$ROOT_DIR:/work" -w /work aquasec/trivy:latest "${TRIVY_CONFIG_FLAGS[@]}"
fi
ok "iac-scan"

# --- container-security (build + trivy image; needs iac + prior jobs conceptually) ---
job "container-security"
docker build -t "pb-backend:${CI_SHA}" pb_backend/
docker build -t "pb-frontend:${CI_SHA}" pb_frontend/

TRIVY_IMAGE_BASE=(image --severity CRITICAL,HIGH --ignore-unfixed --exit-code 1 --vuln-type os,library)
if command -v trivy >/dev/null 2>&1; then
  trivy "${TRIVY_IMAGE_BASE[@]}" --format table "pb-backend:${CI_SHA}"
  trivy "${TRIVY_IMAGE_BASE[@]}" --format table "pb-frontend:${CI_SHA}"
else
  docker run --rm -v /var/run/docker.sock:/var/run/docker.sock aquasec/trivy:latest \
    "${TRIVY_IMAGE_BASE[@]}" --format table "pb-backend:${CI_SHA}"
  docker run --rm -v /var/run/docker.sock:/var/run/docker.sock aquasec/trivy:latest \
    "${TRIVY_IMAGE_BASE[@]}" --format table "pb-frontend:${CI_SHA}"
fi
ok "container-security"

echo -e "\n${GREEN}Local CI completed (parity with .github/workflows/ci.yml).${NC}"
