# kubectl-schemagen demo
# Requires: kind cluster with CRDs installed, kubectl-schemagen in PATH

Output demo.gif
Output demo.mp4

Set Shell bash
Set FontSize 15
Set Width 1000
Set Height 650
Set Theme "Catppuccin Mocha"
Set Padding 20
Set TypingSpeed 40ms

# Title
Type "# kubectl-schemagen: OpenAPI schema-powered Kubernetes tools"
Enter
Sleep 1.5s

# Show available types
Type "kubectl schemagen manifest --list | head -15"
Enter
Sleep 3s

# Generate a Deployment
Type "kubectl schemagen manifest Deployment --name=web --image=myapp:v2 --replicas=3"
Enter
Sleep 4s

# Annotated output
Type "kubectl schemagen manifest Service --name=web --annotate | head -20"
Enter
Sleep 4s

# JSON output
Type "kubectl schemagen manifest Pod --name=debug -o json | jq '.metadata.name'"
Enter
Sleep 3s

# Generate a CRD (Gateway API)
Type "kubectl schemagen manifest HTTPRoute --name=api"
Enter
Sleep 4s

# Detect deprecated APIs
Type "kubectl schemagen migrate old-ingress.yaml || true"
Enter
Sleep 3s

# Scaffold: generate a kustomize base
Type "kubectl schemagen scaffold Deployment Service --name=web -o /tmp/demo-base && ls /tmp/demo-base/"
Enter
Sleep 3s

# Pipe to kubectl apply --dry-run
Type "kubectl schemagen manifest Service --name=web | kubectl apply --dry-run=server -f -"
Enter
Sleep 3s

# Install via krew
Type "kubectl krew install schemagen"
Enter
Sleep 3s

# Done
Type ""
Sleep 2s
