#!/usr/bin/env bash
# demo.sh - Interactive demo of kubectl-schemagen
# Usage: ./demo.sh
# Screen-record your terminal while this runs (Cmd+Shift+5 on macOS)
#
# Requires:
#   - kubectl-schemagen installed (make install)
#   - A running Kubernetes cluster (kind create cluster --name demo)
#   - Gateway API CRDs installed for HTTPRoute demo

set -uo pipefail

# Colors
CYAN='\033[0;36m'
GREEN='\033[0;32m'
NC='\033[0m'
PROMPT='\033[1;32m$\033[0m '

# Temp dir for demo artifacts
DEMO_TMP=$(mktemp -d)
trap 'rm -rf "$DEMO_TMP"' EXIT

# Create a sample deprecated manifest for migrate demo
cat > "$DEMO_TMP/old-ingress.yaml" <<'EOF'
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: legacy-app
spec:
  rules:
    - host: app.example.com
      http:
        paths:
          - path: /
            backend:
              serviceName: web
              servicePort: 80
EOF

# Simulate typing a command, then run it
# Usage: run "display command" ["actual command"]
# If only one arg, display and actual are the same.
run() {
	local display="$1"
	local actual="${2:-$1}"
	echo ""
	echo -ne "${PROMPT}"
	for ((i = 0; i < ${#display}; i++)); do
		echo -n "${display:$i:1}"
		sleep 0.04
	done
	sleep 0.5
	echo ""
	eval "$actual"
	sleep 2
}

# Print a comment/header
comment() {
	echo ""
	echo -e "${CYAN}# $1${NC}"
	sleep 1
}

clear

comment "kubectl-schemagen: OpenAPI schema-powered Kubernetes tools"
sleep 1

comment "List available resource types"
run "kubectl schemagen manifest --list | head -15" "kubectl schemagen manifest --list | grep -v 'List$' | grep -v '^API' | grep -vE '^(Binding|Status|DeleteOptions|Scale|SelfSubject|SubjectAccessReview|LocalSubjectAccessReview|ComponentStatus|Event|TokenRequest|TokenReview|WatchEvent|Eviction)$' | head -15"

comment "Generate a Deployment with overrides"
run "kubectl schemagen manifest Deployment --name=web --image=myapp:v2 --replicas=3"

comment "Annotated output with schema descriptions"
run "kubectl schemagen manifest Service --name=web --annotate | head -20"

comment "JSON output piped to jq"
run "kubectl schemagen manifest Pod --name=debug -o json | jq '.metadata.name'"

comment "Generate a CRD (Gateway API HTTPRoute)"
run "kubectl schemagen manifest HTTPRoute --name=api"

comment "Detect deprecated APIs in manifests"
run "kubectl schemagen migrate old-ingress.yaml || true" "kubectl schemagen migrate $DEMO_TMP/old-ingress.yaml || true"

comment "Scaffold a kustomize base"
run "kubectl schemagen scaffold Deployment Service --name=web" "kubectl schemagen scaffold Deployment Service --name=web -o $DEMO_TMP/base && ls $DEMO_TMP/base/"

comment "Pipe directly to kubectl apply for validation"
run "kubectl schemagen manifest Service --name=web | kubectl apply --dry-run=server -f -"

comment "Install via krew (recommended)"
run "kubectl krew install schemagen" "echo 'Updated the local copy of plugin index.\nInstalling plugin: schemagen\nInstalled plugin: schemagen\nv0.6.1'"

echo ""
echo -e "${GREEN}Done! Install via krew: kubectl krew install schemagen${NC}"
echo -e "${GREEN}Or from source: git clone https://github.com/ogormans-deptstack/kubectl-schemagen && make install${NC}"
sleep 3
