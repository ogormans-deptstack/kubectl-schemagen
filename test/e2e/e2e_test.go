//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var resourceTypes = []struct {
	name      string
	gvr       string
	getKind   string
	extraArgs []string
}{
	{name: "pod", gvr: "v1/Pod", getKind: "pod"},
	{name: "deployment", gvr: "apps/v1/Deployment", getKind: "deployment"},
	{name: "service", gvr: "v1/Service", getKind: "service"},
	{name: "configmap", gvr: "v1/ConfigMap", getKind: "configmap"},
	{name: "secret", gvr: "v1/Secret", getKind: "secret"},
	{name: "job", gvr: "batch/v1/Job", getKind: "job"},
	{name: "cronjob", gvr: "batch/v1/CronJob", getKind: "cronjob"},
	{name: "ingress", gvr: "networking.k8s.io/v1/Ingress", getKind: "ingress"},
	{name: "networkpolicy", gvr: "networking.k8s.io/v1/NetworkPolicy", getKind: "networkpolicy"},
	{name: "statefulset", gvr: "apps/v1/StatefulSet", getKind: "statefulset"},
	{name: "daemonset", gvr: "apps/v1/DaemonSet", getKind: "daemonset"},
	{name: "persistentvolumeclaim", gvr: "v1/PersistentVolumeClaim", getKind: "pvc"},
	{name: "horizontalpodautoscaler", gvr: "autoscaling/v2/HorizontalPodAutoscaler", getKind: "hpa"},
	{name: "role", gvr: "rbac.authorization.k8s.io/v1/Role", getKind: "role"},
	{name: "clusterrole", gvr: "rbac.authorization.k8s.io/v1/ClusterRole", getKind: "clusterrole"},
	{name: "rolebinding", gvr: "rbac.authorization.k8s.io/v1/RoleBinding", getKind: "rolebinding"},
	{name: "clusterrolebinding", gvr: "rbac.authorization.k8s.io/v1/ClusterRoleBinding", getKind: "clusterrolebinding"},
	{name: "serviceaccount", gvr: "v1/ServiceAccount", getKind: "serviceaccount"},
	{name: "resourcequota", gvr: "v1/ResourceQuota", getKind: "resourcequota"},
	{name: "limitrange", gvr: "v1/LimitRange", getKind: "limitrange"},
	{name: "persistentvolume", gvr: "v1/PersistentVolume", getKind: "pv"},
	{name: "poddisruptionbudget", gvr: "policy/v1/PodDisruptionBudget", getKind: "pdb"},
	{name: "ingressclass", gvr: "networking.k8s.io/v1/IngressClass", getKind: "ingressclass"},
	{name: "storageclass", gvr: "storage.k8s.io/v1/StorageClass", getKind: "storageclass"},
	{name: "priorityclass", gvr: "scheduling.k8s.io/v1/PriorityClass", getKind: "priorityclass"},
	{name: "runtimeclass", gvr: "node.k8s.io/v1/RuntimeClass", getKind: "runtimeclass"},
}

var dryRunOnlyTypes = []struct {
	name    string
	getKind string
}{
	{name: "namespace", getKind: "namespace"},
	{name: "validatingwebhookconfiguration", getKind: "validatingwebhookconfiguration"},
	{name: "mutatingwebhookconfiguration", getKind: "mutatingwebhookconfiguration"},
	{name: "customresourcedefinition", getKind: "customresourcedefinition"},
}

var crdTypes = []struct {
	name      string
	gvr       string
	getKind   string
	extraArgs []string
}{
	{name: "crontab", gvr: "stable.example.com/v1/CronTab", getKind: "crontab"},
}

var gatewayTypes = []struct {
	name    string
	getKind string
}{
	{name: "HTTPRoute", getKind: "httproute"},
	{name: "Gateway", getKind: "gateway"},
	{name: "GatewayClass", getKind: "gatewayclass"},
	{name: "GRPCRoute", getKind: "grpcroute"},
	{name: "TCPRoute", getKind: "tcproute"},
	{name: "TLSRoute", getKind: "tlsroute"},
	{name: "UDPRoute", getKind: "udproute"},
	{name: "ReferenceGrant", getKind: "referencegrant"},
	{name: "BackendLBPolicy", getKind: "backendlbpolicy"},
	{name: "BackendTLSPolicy", getKind: "backendtlspolicy"},
}

var argoTypes = []struct {
	name    string
	getKind string
}{
	{name: "Workflow", getKind: "workflow"},
	{name: "CronWorkflow", getKind: "cronworkflow"},
	{name: "WorkflowTemplate", getKind: "workflowtemplate"},
	{name: "ClusterWorkflowTemplate", getKind: "clusterworkflowtemplate"},
}

var certManagerTypes = []struct {
	name    string
	getKind string
}{
	{name: "Certificate", getKind: "certificate"},
	{name: "Issuer", getKind: "issuer"},
	{name: "ClusterIssuer", getKind: "clusterissuer"},
}

var crossplaneTypes = []struct {
	name    string
	getKind string
}{
	{name: "Composition", getKind: "composition"},
	{name: "CompositeResourceDefinition", getKind: "compositeresourcedefinition"},
	{name: "EnvironmentConfig", getKind: "environmentconfig"},
}

func TestGenerateAndCreate(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)

	for _, rt := range resourceTypes {
		t.Run(rt.name, func(t *testing.T) {
			cleanupResource(t, rt.getKind, rt.name)

			t.Run("generates valid YAML", func(t *testing.T) {
				yaml := runManifest(t, binaryPath, rt.name, rt.extraArgs...)
				assertValidYAML(t, yaml)
				assertContainsKind(t, yaml, rt.name)
			})

			t.Run("server dry-run validates", func(t *testing.T) {
				yaml := runManifest(t, binaryPath, rt.name, rt.extraArgs...)
				kubectlDryRun(t, yaml)
			})

			t.Run("creates resource via kubectl create", func(t *testing.T) {
				yaml := runManifest(t, binaryPath, rt.name, rt.extraArgs...)
				kubectlCreate(t, yaml)
				assertResourceExists(t, rt.getKind)
			})
		})
	}
}

func TestDryRunOnlyGenerateAndValidate(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)

	for _, rt := range dryRunOnlyTypes {
		t.Run(rt.name, func(t *testing.T) {
			t.Run("generates valid YAML", func(t *testing.T) {
				yaml := runManifest(t, binaryPath, rt.name)
				assertValidYAML(t, yaml)
				assertContainsKind(t, yaml, rt.name)
			})

			t.Run("server dry-run validates", func(t *testing.T) {
				yaml := runManifest(t, binaryPath, rt.name)
				kubectlDryRun(t, yaml)
			})
		})
	}
}

func TestDynamicFlags(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)

	t.Run("deployment respects --name flag", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "deployment", "--name=custom-app")
		assertYAMLContains(t, yaml, "name: custom-app")
	})

	t.Run("deployment respects --replicas flag", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "deployment", "--replicas=5")
		assertYAMLContains(t, yaml, "replicas: 5")
	})

	t.Run("pod respects --image flag", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "pod", "--image=nginx:latest")
		assertYAMLContains(t, yaml, "nginx:latest")
	})
}

func TestSpecNuances(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)

	t.Run("service includes ports spec", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "service")
		assertYAMLContains(t, yaml, "ports:")
	})

	t.Run("statefulset includes serviceName", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "statefulset")
		assertYAMLContains(t, yaml, "serviceName:")
	})

	t.Run("ingress includes rules", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "ingress")
		assertYAMLContains(t, yaml, "rules:")
	})

	t.Run("cronjob includes schedule", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "cronjob")
		assertYAMLContains(t, yaml, "schedule:")
	})

	t.Run("hpa includes metrics", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "horizontalpodautoscaler")
		assertYAMLContains(t, yaml, "metrics:")
	})

	t.Run("pvc includes access modes and resources", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "persistentvolumeclaim")
		assertYAMLContains(t, yaml, "accessModes:")
		assertYAMLContains(t, yaml, "resources:")
	})
}

func TestNativeResourceSpecNuances(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)

	t.Run("role contains rules", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "role")
		assertYAMLContains(t, yaml, "rules:")
	})

	t.Run("clusterrole contains rules", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "clusterrole")
		assertYAMLContains(t, yaml, "rules:")
	})

	t.Run("rolebinding contains roleRef and subjects", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "rolebinding")
		assertYAMLContains(t, yaml, "roleRef:")
		assertYAMLContains(t, yaml, "subjects:")
	})

	t.Run("clusterrolebinding contains roleRef and subjects", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "clusterrolebinding")
		assertYAMLContains(t, yaml, "roleRef:")
		assertYAMLContains(t, yaml, "subjects:")
	})

	t.Run("poddisruptionbudget contains disruption settings", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "poddisruptionbudget")
		if !strings.Contains(yaml, "minAvailable:") && !strings.Contains(yaml, "maxUnavailable:") {
			t.Errorf("PodDisruptionBudget YAML missing both minAvailable and maxUnavailable\ngot:\n%s", yaml)
		}
	})

	t.Run("resourcequota contains hard", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "resourcequota")
		assertYAMLContains(t, yaml, "hard:")
	})

	t.Run("limitrange contains limits", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "limitrange")
		assertYAMLContains(t, yaml, "limits:")
	})

	t.Run("persistentvolume contains capacity and hostPath", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "persistentvolume")
		assertYAMLContains(t, yaml, "capacity:")
		assertYAMLContains(t, yaml, "hostPath:")
	})

	t.Run("storageclass contains provisioner", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "storageclass")
		assertYAMLContains(t, yaml, "provisioner:")
	})

	t.Run("priorityclass contains value", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "priorityclass")
		assertYAMLContains(t, yaml, "value:")
	})

	t.Run("runtimeclass contains handler", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "runtimeclass")
		assertYAMLContains(t, yaml, "handler:")
	})

	t.Run("validatingwebhookconfiguration contains webhooks and admissionReviewVersions", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "validatingwebhookconfiguration")
		assertYAMLContains(t, yaml, "webhooks:")
		assertYAMLContains(t, yaml, "admissionReviewVersions:")
	})

	t.Run("mutatingwebhookconfiguration contains webhooks and admissionReviewVersions", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "mutatingwebhookconfiguration")
		assertYAMLContains(t, yaml, "webhooks:")
		assertYAMLContains(t, yaml, "admissionReviewVersions:")
	})

	t.Run("customresourcedefinition contains group names and versions", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "customresourcedefinition")
		assertYAMLContains(t, yaml, "group:")
		assertYAMLContains(t, yaml, "names:")
		assertYAMLContains(t, yaml, "versions:")
	})

	t.Run("ingressclass contains controller", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "ingressclass")
		assertYAMLContains(t, yaml, "controller:")
	})
}

func TestCRDGenerateAndCreate(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)
	ensureCRD(t)

	for _, rt := range crdTypes {
		t.Run(rt.name, func(t *testing.T) {
			cleanupResource(t, rt.getKind, rt.name)

			t.Run("generates valid YAML", func(t *testing.T) {
				yaml := runManifest(t, binaryPath, rt.name, rt.extraArgs...)
				assertValidYAML(t, yaml)
				assertContainsKind(t, yaml, rt.name)
			})

			t.Run("server dry-run validates", func(t *testing.T) {
				yaml := runManifest(t, binaryPath, rt.name, rt.extraArgs...)
				kubectlDryRun(t, yaml)
			})

			t.Run("creates resource via kubectl create", func(t *testing.T) {
				yaml := runManifest(t, binaryPath, rt.name, rt.extraArgs...)
				kubectlCreate(t, yaml)
				assertResourceExists(t, rt.getKind)
			})
		})
	}
}

func TestCRDSpecNuances(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)
	ensureCRD(t)

	t.Run("crontab includes required field cronSpec", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "crontab")
		assertYAMLContains(t, yaml, "cronSpec:")
	})

	t.Run("crontab includes required field image", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "crontab")
		assertYAMLContains(t, yaml, "image:")
	})

	t.Run("crontab includes optional replicas", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "crontab")
		assertYAMLContains(t, yaml, "replicas:")
	})

	t.Run("crontab includes restartPolicy enum value", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "crontab")
		assertYAMLContains(t, yaml, "restartPolicy:")
	})

	t.Run("crontab has correct apiVersion", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "crontab")
		assertYAMLContains(t, yaml, "apiVersion: stable.example.com/v1")
	})

	t.Run("crontab respects --set cronSpec override", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "crontab", "--set", "cronSpec=*/15 * * * *")
		assertYAMLContains(t, yaml, "cronSpec:")
	})

	t.Run("crontab respects --image override", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "crontab", "--image=redis:7")
		assertYAMLContains(t, yaml, "redis:7")
	})
}

func TestCRDDiscovery(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)
	ensureCRD(t)

	t.Run("list includes CronTab from CRD", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "manifest", "--list")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("--list failed: %v", err)
		}
		output := strings.ToLower(string(out))
		if !strings.Contains(output, "crontab") {
			t.Errorf("--list output missing CRD type CronTab\ngot:\n%s", string(out))
		}
	})
}

func TestGatewayAPIGenerateAndCreate(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)
	ensureGatewayCRDs(t)

	for _, rt := range gatewayTypes {
		t.Run(rt.name, func(t *testing.T) {
			cleanupResource(t, rt.getKind, strings.ToLower(rt.name))

			t.Run("generates valid YAML", func(t *testing.T) {
				yaml := runManifest(t, binaryPath, rt.name)
				assertValidYAML(t, yaml)
			})

			t.Run("server dry-run validates", func(t *testing.T) {
				yaml := runManifest(t, binaryPath, rt.name)
				kubectlDryRun(t, yaml)
			})

			t.Run("creates resource via kubectl apply", func(t *testing.T) {
				yaml := runManifest(t, binaryPath, rt.name)
				kubectlApply(t, yaml)
				assertResourceExists(t, rt.getKind)
			})
		})
	}
}

func TestGatewayAPISpecNuances(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)
	ensureGatewayCRDs(t)

	t.Run("HTTPRoute includes parentRefs", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "HTTPRoute")
		assertYAMLContains(t, yaml, "parentRefs:")
	})

	t.Run("HTTPRoute includes rules with backendRefs", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "HTTPRoute")
		assertYAMLContains(t, yaml, "rules:")
		assertYAMLContains(t, yaml, "backendRefs:")
	})

	t.Run("HTTPRoute filter has type and matching sibling", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "HTTPRoute")
		assertYAMLContains(t, yaml, "filters:")
		assertYAMLContains(t, yaml, "type: RequestHeaderModifier")
		assertYAMLContains(t, yaml, "requestHeaderModifier:")
	})

	t.Run("GatewayClass includes controllerName with domain format", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "GatewayClass")
		assertYAMLContains(t, yaml, "controllerName: example.com/example")
	})

	t.Run("Gateway includes listeners", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "Gateway")
		assertYAMLContains(t, yaml, "listeners:")
		assertYAMLContains(t, yaml, "gatewayClassName:")
	})

	t.Run("Gateway has no addresses field", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "Gateway")
		if strings.Contains(yaml, "addresses:") {
			t.Error("Gateway should not include addresses (has oneOf)")
		}
	})

	t.Run("BackendTLSPolicy includes validation with hostname", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "BackendTLSPolicy")
		assertYAMLContains(t, yaml, "validation:")
		assertYAMLContains(t, yaml, "hostname:")
	})

	t.Run("BackendTLSPolicy subjectAltNames resolves discriminated union", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "BackendTLSPolicy")
		assertYAMLContains(t, yaml, "subjectAltNames:")
		assertYAMLContains(t, yaml, "type: Hostname")
	})

	t.Run("ReferenceGrant has correct apiVersion", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "ReferenceGrant")
		assertYAMLContains(t, yaml, "apiVersion: gateway.networking.k8s.io/")
	})
}

func TestGatewayAPIDiscovery(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)
	ensureGatewayCRDs(t)

	t.Run("list includes Gateway API types", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "manifest", "--list")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("--list failed: %v", err)
		}
		output := strings.ToLower(string(out))
		for _, rt := range gatewayTypes {
			if !strings.Contains(output, strings.ToLower(rt.name)) {
				t.Errorf("--list output missing Gateway API type: %s\ngot:\n%s", rt.name, string(out))
			}
		}
	})
}

func TestArgoGenerateAndCreate(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)
	ensureArgoCRDs(t)

	for _, rt := range argoTypes {
		t.Run(rt.name, func(t *testing.T) {
			cleanupResource(t, rt.getKind, strings.ToLower(rt.name))

			t.Run("generates valid YAML", func(t *testing.T) {
				yaml := runManifest(t, binaryPath, rt.name)
				assertValidYAML(t, yaml)
			})

			t.Run("server dry-run validates", func(t *testing.T) {
				yaml := runManifest(t, binaryPath, rt.name)
				kubectlDryRun(t, yaml)
			})

			t.Run("creates resource via kubectl apply", func(t *testing.T) {
				yaml := runManifest(t, binaryPath, rt.name)
				kubectlApply(t, yaml)
				assertResourceExists(t, rt.getKind)
			})
		})
	}
}

func TestArgoSpecNuances(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)
	ensureArgoCRDs(t)

	t.Run("Workflow includes templates array", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "Workflow")
		assertYAMLContains(t, yaml, "templates:")
	})

	t.Run("Workflow has correct apiVersion", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "Workflow")
		assertYAMLContains(t, yaml, "apiVersion: argoproj.io/")
	})

	t.Run("CronWorkflow includes schedules", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "CronWorkflow")
		assertYAMLContains(t, yaml, "schedules:")
	})

	t.Run("WorkflowTemplate includes templates", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "WorkflowTemplate")
		assertYAMLContains(t, yaml, "templates:")
	})
}

func TestArgoDiscovery(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)
	ensureArgoCRDs(t)

	t.Run("list includes Argo types", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "manifest", "--list")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("--list failed: %v", err)
		}
		output := strings.ToLower(string(out))
		for _, rt := range argoTypes {
			if !strings.Contains(output, strings.ToLower(rt.name)) {
				t.Errorf("--list output missing Argo type: %s\ngot:\n%s", rt.name, string(out))
			}
		}
	})
}

func TestCertManagerGenerateAndCreate(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)
	ensureCertManagerCRDs(t)

	for _, rt := range certManagerTypes {
		t.Run(rt.name, func(t *testing.T) {
			cleanupResource(t, rt.getKind, strings.ToLower(rt.name))

			t.Run("generates valid YAML", func(t *testing.T) {
				yaml := runManifest(t, binaryPath, rt.name)
				assertValidYAML(t, yaml)
			})

			t.Run("server dry-run validates", func(t *testing.T) {
				yaml := runManifest(t, binaryPath, rt.name)
				kubectlDryRun(t, yaml)
			})

			t.Run("creates resource via kubectl apply", func(t *testing.T) {
				yaml := runManifest(t, binaryPath, rt.name)
				kubectlApply(t, yaml)
				assertResourceExists(t, rt.getKind)
			})
		})
	}
}

func TestCertManagerSpecNuances(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)
	ensureCertManagerCRDs(t)

	t.Run("Certificate includes secretName", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "Certificate")
		assertYAMLContains(t, yaml, "secretName:")
	})

	t.Run("Certificate includes issuerRef", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "Certificate")
		assertYAMLContains(t, yaml, "issuerRef:")
	})

	t.Run("Certificate has correct apiVersion", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "Certificate")
		assertYAMLContains(t, yaml, "apiVersion: cert-manager.io/")
	})

	t.Run("Issuer has correct apiVersion", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "Issuer")
		assertYAMLContains(t, yaml, "apiVersion: cert-manager.io/")
	})

	t.Run("ClusterIssuer has correct apiVersion", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "ClusterIssuer")
		assertYAMLContains(t, yaml, "apiVersion: cert-manager.io/")
	})
}

func TestCertManagerDiscovery(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)
	ensureCertManagerCRDs(t)

	t.Run("list includes cert-manager types", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "manifest", "--list")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("--list failed: %v", err)
		}
		output := strings.ToLower(string(out))
		for _, rt := range certManagerTypes {
			if !strings.Contains(output, strings.ToLower(rt.name)) {
				t.Errorf("--list output missing cert-manager type: %s\ngot:\n%s", rt.name, string(out))
			}
		}
	})
}

func TestCrossplaneGenerateAndCreate(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)
	ensureCrossplaneCRDs(t)

	for _, rt := range crossplaneTypes {
		t.Run(rt.name, func(t *testing.T) {
			cleanupResource(t, rt.getKind, strings.ToLower(rt.name))

			t.Run("generates valid YAML", func(t *testing.T) {
				yaml := runManifest(t, binaryPath, rt.name)
				assertValidYAML(t, yaml)
			})

			t.Run("server dry-run validates", func(t *testing.T) {
				yaml := runManifest(t, binaryPath, rt.name)
				kubectlDryRun(t, yaml)
			})

			t.Run("creates resource via kubectl apply", func(t *testing.T) {
				yaml := runManifest(t, binaryPath, rt.name)
				kubectlApply(t, yaml)
				assertResourceExists(t, rt.getKind)
			})
		})
	}
}

func TestCrossplaneSpecNuances(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)
	ensureCrossplaneCRDs(t)

	t.Run("Composition has correct apiVersion", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "Composition")
		assertYAMLContains(t, yaml, "apiVersion: apiextensions.crossplane.io/")
	})

	t.Run("CompositeResourceDefinition has correct apiVersion", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "CompositeResourceDefinition")
		assertYAMLContains(t, yaml, "apiVersion: apiextensions.crossplane.io/")
	})

	t.Run("Composition includes resources or pipeline", func(t *testing.T) {
		yaml := runManifest(t, binaryPath, "Composition")
		if !strings.Contains(yaml, "resources:") && !strings.Contains(yaml, "pipeline:") {
			t.Errorf("Composition YAML missing both resources and pipeline fields\ngot:\n%s", yaml)
		}
	})
}

func TestCrossplaneDiscovery(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)
	ensureCrossplaneCRDs(t)

	t.Run("list includes Crossplane types", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "manifest", "--list")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("--list failed: %v", err)
		}
		output := strings.ToLower(string(out))
		for _, rt := range crossplaneTypes {
			if !strings.Contains(output, strings.ToLower(rt.name)) {
				t.Errorf("--list output missing Crossplane type: %s\ngot:\n%s", rt.name, string(out))
			}
		}
	})
}

func TestOpenAPISpecResilience(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)

	t.Run("handles unknown resource type gracefully", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "manifest", "nonexistentresource")
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err == nil {
			t.Fatal("expected error for unknown resource type")
		}
		if !strings.Contains(stderr.String(), "nonexistentresource") {
			t.Errorf("error should mention the resource type, got: %s", stderr.String())
		}
	})

	t.Run("list shows at least the core types", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "manifest", "--list")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("--list failed: %v", err)
		}
		output := string(out)
		for _, rt := range resourceTypes {
			if !strings.Contains(strings.ToLower(output), rt.name) {
				t.Errorf("--list output missing resource type: %s", rt.name)
			}
		}
	})
}

func findBinary(t *testing.T) string {
	t.Helper()
	path := "../../bin/kubectl-schemagen"
	if _, err := exec.LookPath(path); err != nil {
		path = "kubectl-schemagen"
		if _, err := exec.LookPath(path); err != nil {
			t.Skip("kubectl-schemagen binary not found; run 'make build' first")
		}
	}
	return path
}

func ensureCluster(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", "cluster-info")
	if err := cmd.Run(); err != nil {
		t.Skip("no cluster available; start a kind cluster first")
	}
}

func ensureCRD(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", "get", "crd", "crontabs.stable.example.com")
	if err := cmd.Run(); err != nil {
		t.Skip("CRD crontabs.stable.example.com not installed; install test CRD first")
	}
}

func ensureGatewayCRDs(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", "get", "crd", "httproutes.gateway.networking.k8s.io")
	if err := cmd.Run(); err != nil {
		t.Skip("Gateway API CRDs not installed; install with: kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/experimental-install.yaml")
	}
}

func ensureArgoCRDs(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", "get", "crd", "workflows.argoproj.io")
	if err := cmd.Run(); err != nil {
		t.Skip("Argo Workflows CRDs not installed; install with: kubectl apply -f https://github.com/argoproj/argo-workflows/releases/download/v4.0.4/install.yaml")
	}
}

func ensureCertManagerCRDs(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", "get", "crd", "certificates.cert-manager.io")
	if err := cmd.Run(); err != nil {
		t.Skip("cert-manager CRDs not installed; install with: kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.20.2/cert-manager.crds.yaml")
	}
}

func ensureCrossplaneCRDs(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", "get", "crd", "compositions.apiextensions.crossplane.io")
	if err := cmd.Run(); err != nil {
		t.Skip("Crossplane CRDs not installed; Crossplane CRDs need manual install via Helm")
	}
}

func runManifest(t *testing.T, binary string, resourceType string, extraArgs ...string) string {
	t.Helper()
	args := append([]string{"manifest", resourceType}, extraArgs...)
	cmd := exec.Command(binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("kubectl-schemagen manifest %s failed: %v\nstderr: %s", resourceType, err, stderr.String())
	}
	return stdout.String()
}

func assertValidYAML(t *testing.T, yaml string) {
	t.Helper()
	if len(strings.TrimSpace(yaml)) == 0 {
		t.Fatal("generated YAML is empty")
	}
	if !strings.Contains(yaml, "apiVersion:") {
		t.Error("YAML missing apiVersion")
	}
	if !strings.Contains(yaml, "kind:") {
		t.Error("YAML missing kind")
	}
	if !strings.Contains(yaml, "metadata:") {
		t.Error("YAML missing metadata")
	}
}

func assertContainsKind(t *testing.T, yaml string, resourceType string) {
	t.Helper()
	if !strings.Contains(strings.ToLower(yaml), "kind:") {
		t.Errorf("YAML for %s missing kind field", resourceType)
	}
}

func assertYAMLContains(t *testing.T, yaml string, substr string) {
	t.Helper()
	if !strings.Contains(yaml, substr) {
		t.Errorf("YAML missing expected content %q\ngot:\n%s", substr, yaml)
	}
}

func kubectlCreate(t *testing.T, yaml string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", "create", "-f", "-")
	cmd.Stdin = strings.NewReader(yaml)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("kubectl create failed: %v\nstderr: %s\nyaml:\n%s", err, stderr.String(), yaml)
	}
}

func kubectlDryRun(t *testing.T, yaml string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", "create", "--dry-run=server", "-f", "-")
	cmd.Stdin = strings.NewReader(yaml)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("kubectl create --dry-run=server failed: %v\nstderr: %s\nyaml:\n%s", err, stderr.String(), yaml)
	}
}

func kubectlApply(t *testing.T, yaml string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(yaml)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("kubectl apply failed: %v\nstderr: %s\nyaml:\n%s", err, stderr.String(), yaml)
	}
}

func assertResourceExists(t *testing.T, kind string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", "get", kind, "--no-headers")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("kubectl get %s failed: %v", kind, err)
	}
	if len(strings.TrimSpace(string(out))) == 0 {
		t.Errorf("no %s resources found after create", kind)
	}
}

func cleanupResource(t *testing.T, kind, name string) {
	t.Helper()
	resourceName := fmt.Sprintf("example-%s", name)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", "delete", kind, resourceName,
		"--ignore-not-found", "--grace-period=0", "--force")
	_ = cmd.Run()
	for range 30 {
		check := exec.CommandContext(ctx, "kubectl", "get", kind, resourceName, "--no-headers")
		if err := check.Run(); err != nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func TestRootCommand(t *testing.T) {
	binaryPath := findBinary(t)

	t.Run("version flag outputs version string", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "--version")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("--version failed: %v", err)
		}
		output := string(out)
		if !strings.Contains(output, "kubectl-schemagen version") {
			t.Errorf("unexpected version output: %s", output)
		}
	})

	t.Run("help lists all subcommands", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "--help")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("--help failed: %v", err)
		}
		output := string(out)
		for _, sub := range []string{"manifest", "migrate", "scaffold"} {
			if !strings.Contains(output, sub) {
				t.Errorf("help output missing subcommand %q\ngot:\n%s", sub, output)
			}
		}
	})

	t.Run("no args shows help without error", func(t *testing.T) {
		cmd := exec.Command(binaryPath)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("root command with no args failed: %v\nstderr: %s", err, stderr.String())
		}
		if !strings.Contains(string(out), "manifest") {
			t.Error("expected help output with subcommand listing")
		}
	})
}

func TestMigrateCurrentAPIs(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)

	t.Run("reports OK for current Deployment apiVersion", func(t *testing.T) {
		manifest := "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: test\n"
		file := writeTestManifest(t, "deploy.yaml", manifest)

		out, err := runMigrate(binaryPath, file)
		if err != nil {
			t.Fatalf("migrate failed: %v\noutput: %s", err, out)
		}
		if !strings.Contains(out, "OK") {
			t.Errorf("expected OK status for apps/v1 Deployment\ngot: %s", out)
		}
	})

	t.Run("reports OK for current Service apiVersion", func(t *testing.T) {
		manifest := "apiVersion: v1\nkind: Service\nmetadata:\n  name: test\n"
		file := writeTestManifest(t, "svc.yaml", manifest)

		out, err := runMigrate(binaryPath, file)
		if err != nil {
			t.Fatalf("migrate failed: %v\noutput: %s", err, out)
		}
		if !strings.Contains(out, "OK") {
			t.Errorf("expected OK status for v1 Service\ngot: %s", out)
		}
	})

	t.Run("handles multi-document YAML", func(t *testing.T) {
		manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
---
apiVersion: v1
kind: Service
metadata:
  name: web
`
		file := writeTestManifest(t, "multi.yaml", manifest)

		out, err := runMigrate(binaryPath, file)
		if err != nil {
			t.Fatalf("migrate failed: %v\noutput: %s", err, out)
		}
		if strings.Count(out, "OK") < 2 {
			t.Errorf("expected at least 2 OK statuses for multi-doc\ngot: %s", out)
		}
	})

	t.Run("handles multiple files", func(t *testing.T) {
		deploy := writeTestManifest(t, "deploy.yaml", "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: test\n")
		svc := writeTestManifest(t, "svc.yaml", "apiVersion: v1\nkind: Service\nmetadata:\n  name: test\n")

		cmd := exec.Command(binaryPath, "migrate", deploy, svc)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			t.Fatalf("migrate failed: %v\nstderr: %s", err, stderr.String())
		}
		if strings.Count(stdout.String(), "OK") < 2 {
			t.Errorf("expected OK for both files\ngot: %s", stdout.String())
		}
	})
}

func TestMigrateDeprecatedAPIs(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)

	t.Run("detects deprecated extensions/v1beta1 Ingress", func(t *testing.T) {
		manifest := "apiVersion: extensions/v1beta1\nkind: Ingress\nmetadata:\n  name: test\n"
		file := writeTestManifest(t, "ingress.yaml", manifest)

		out, exitErr := runMigrate(binaryPath, file)
		if exitErr == nil {
			t.Log("migrate did not return error for deprecated API (may be removed, not deprecated)")
		}
		if !strings.Contains(out, "DEP") && !strings.Contains(out, "REM") {
			t.Errorf("expected DEP or REM status for extensions/v1beta1 Ingress\ngot: %s", out)
		}
	})

	t.Run("detects removed apps/v1beta1 Deployment", func(t *testing.T) {
		manifest := "apiVersion: apps/v1beta1\nkind: Deployment\nmetadata:\n  name: test\n"
		file := writeTestManifest(t, "deploy-old.yaml", manifest)

		out, exitErr := runMigrate(binaryPath, file)
		if exitErr == nil {
			t.Log("migrate did not return error for removed API")
		}
		if !strings.Contains(out, "DEP") && !strings.Contains(out, "REM") {
			t.Errorf("expected DEP or REM for apps/v1beta1 Deployment\ngot: %s", out)
		}
	})
}

func TestMigrateEdgeCases(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)

	t.Run("requires at least one file argument", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "migrate")
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err == nil {
			t.Fatal("expected error when no files provided")
		}
	})

	t.Run("fails on nonexistent file", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "migrate", "/tmp/does-not-exist.yaml")
		err := cmd.Run()
		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
	})

	t.Run("handles empty documents gracefully", func(t *testing.T) {
		manifest := "---\n---\n"
		file := writeTestManifest(t, "empty.yaml", manifest)

		cmd := exec.Command(binaryPath, "migrate", file)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		_ = cmd.Run()
	})
}

func TestManifestJSONOutput(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)

	t.Run("generates valid JSON with -o json", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "manifest", "Deployment", "-o", "json")
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("manifest -o json failed: %v\nstderr: %s", err, stderr.String())
		}
		output := stdout.String()
		if !strings.HasPrefix(strings.TrimSpace(output), "{") {
			t.Errorf("expected JSON object, got: %.50s", output)
		}
		if !strings.Contains(output, `"kind"`) {
			t.Error("JSON output missing kind field")
		}
		if !strings.Contains(output, `"Deployment"`) {
			t.Error("JSON output missing Deployment kind value")
		}
	})

	t.Run("JSON output passes dry-run", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "manifest", "ConfigMap", "-o", "json", "--name=json-test-cm")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			t.Fatalf("manifest -o json failed: %v", err)
		}
		// kubectl create accepts JSON directly
		kubectlDryRun(t, stdout.String())
	})

	t.Run("rejects invalid output format", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "manifest", "Pod", "-o", "xml")
		err := cmd.Run()
		if err == nil {
			t.Fatal("expected error for unsupported output format")
		}
	})
}

func TestMigrateJSONOutput(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)

	t.Run("outputs JSON with -o json", func(t *testing.T) {
		manifest := "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: test\n"
		file := writeTestManifest(t, "deploy-json.yaml", manifest)

		cmd := exec.Command(binaryPath, "migrate", "-o", "json", file)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("migrate -o json failed: %v\nstderr: %s", err, stderr.String())
		}
		output := stdout.String()
		if !strings.HasPrefix(strings.TrimSpace(output), "[") {
			t.Errorf("expected JSON array, got: %.50s", output)
		}
		if !strings.Contains(output, `"status"`) {
			t.Error("JSON output missing status field")
		}
		if !strings.Contains(output, `"OK"`) {
			t.Error("JSON output missing OK status")
		}
	})

	t.Run("JSON output includes file and kind", func(t *testing.T) {
		manifest := "apiVersion: v1\nkind: Service\nmetadata:\n  name: web\n"
		file := writeTestManifest(t, "svc-json.yaml", manifest)

		cmd := exec.Command(binaryPath, "migrate", "-o", "json", file)
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			t.Fatalf("migrate -o json failed: %v", err)
		}
		output := stdout.String()
		if !strings.Contains(output, `"kind": "Service"`) {
			t.Errorf("JSON missing kind field\ngot: %s", output)
		}
		if !strings.Contains(output, `"name": "web"`) {
			t.Errorf("JSON missing name field\ngot: %s", output)
		}
	})
}

func TestMigrateStdin(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)

	t.Run("reads from stdin with dash argument", func(t *testing.T) {
		manifest := "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: test\n"
		cmd := exec.Command(binaryPath, "migrate", "-")
		cmd.Stdin = strings.NewReader(manifest)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("migrate - failed: %v\nstderr: %s", err, stderr.String())
		}
		output := stdout.String()
		if !strings.Contains(output, "OK") {
			t.Errorf("expected OK status from stdin\ngot: %s", output)
		}
		if !strings.Contains(output, "<stdin>") {
			t.Errorf("expected <stdin> in file path\ngot: %s", output)
		}
	})
}

func TestMigrateExitCodes(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)

	t.Run("exit 0 for current APIs", func(t *testing.T) {
		manifest := "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: test\n"
		file := writeTestManifest(t, "current.yaml", manifest)

		cmd := exec.Command(binaryPath, "migrate", file)
		err := cmd.Run()
		if err != nil {
			t.Errorf("expected exit 0 for current APIs, got: %v", err)
		}
	})

	t.Run("non-zero exit for removed APIs", func(t *testing.T) {
		manifest := "apiVersion: extensions/v1beta1\nkind: Deployment\nmetadata:\n  name: test\n"
		file := writeTestManifest(t, "removed.yaml", manifest)

		cmd := exec.Command(binaryPath, "migrate", file)
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		err := cmd.Run()
		if err == nil {
			t.Fatal("expected non-zero exit for removed APIs")
		}
	})
}

func TestScaffoldSingleResource(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)

	t.Run("creates output directory and files", func(t *testing.T) {
		outDir := filepath.Join(t.TempDir(), "base")

		cmd := exec.Command(binaryPath, "scaffold", "Deployment", "-o", outDir)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("scaffold failed: %v\nstderr: %s", err, stderr.String())
		}

		deployPath := filepath.Join(outDir, "deployment.yaml")
		data, err := os.ReadFile(deployPath)
		if err != nil {
			t.Fatalf("deployment.yaml not created: %v", err)
		}
		content := string(data)
		if !strings.Contains(content, "apiVersion:") || !strings.Contains(content, "kind:") {
			t.Errorf("deployment.yaml missing apiVersion/kind\ngot:\n%s", content)
		}

		kPath := filepath.Join(outDir, "kustomization.yaml")
		kData, err := os.ReadFile(kPath)
		if err != nil {
			t.Fatalf("kustomization.yaml not created: %v", err)
		}
		kContent := string(kData)
		if !strings.Contains(kContent, "deployment.yaml") {
			t.Errorf("kustomization.yaml missing deployment.yaml reference\ngot:\n%s", kContent)
		}
		if !strings.Contains(kContent, "apiVersion: kustomize.config.k8s.io/v1beta1") {
			t.Errorf("kustomization.yaml missing apiVersion\ngot:\n%s", kContent)
		}
	})

	t.Run("generated manifest passes dry-run", func(t *testing.T) {
		outDir := filepath.Join(t.TempDir(), "base")

		cmd := exec.Command(binaryPath, "scaffold", "Deployment", "--name=scaffold-dryrun-test", "-o", outDir)
		if err := cmd.Run(); err != nil {
			t.Fatalf("scaffold failed: %v", err)
		}

		data, err := os.ReadFile(filepath.Join(outDir, "deployment.yaml"))
		if err != nil {
			t.Fatalf("read deployment.yaml: %v", err)
		}
		kubectlDryRun(t, string(data))
	})
}

func TestScaffoldMultipleResources(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)

	t.Run("scaffolds multiple resources into one directory", func(t *testing.T) {
		outDir := filepath.Join(t.TempDir(), "multi")

		cmd := exec.Command(binaryPath, "scaffold", "Deployment", "Service", "-o", outDir)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("scaffold failed: %v\nstderr: %s", err, stderr.String())
		}

		for _, file := range []string{"deployment.yaml", "service.yaml"} {
			if _, err := os.Stat(filepath.Join(outDir, file)); os.IsNotExist(err) {
				t.Errorf("expected %s to be created", file)
			}
		}

		kData, _ := os.ReadFile(filepath.Join(outDir, "kustomization.yaml"))
		kContent := string(kData)
		if !strings.Contains(kContent, "deployment.yaml") || !strings.Contains(kContent, "service.yaml") {
			t.Errorf("kustomization.yaml missing resource references\ngot:\n%s", kContent)
		}
	})
}

func TestScaffoldWithOverrides(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)

	t.Run("respects --name flag", func(t *testing.T) {
		outDir := filepath.Join(t.TempDir(), "named")

		cmd := exec.Command(binaryPath, "scaffold", "Deployment", "--name=myapp", "-o", outDir)
		if err := cmd.Run(); err != nil {
			t.Fatalf("scaffold with --name failed: %v", err)
		}

		data, _ := os.ReadFile(filepath.Join(outDir, "deployment.yaml"))
		if !strings.Contains(string(data), "name: myapp") {
			t.Errorf("scaffold did not apply --name override\ngot:\n%s", string(data))
		}
	})

	t.Run("requires at least one resource type", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "scaffold")
		err := cmd.Run()
		if err == nil {
			t.Fatal("expected error when no resource type provided")
		}
	})
}

func TestScaffoldKustomizeBuild(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)

	t.Run("kustomize build succeeds on scaffolded output", func(t *testing.T) {
		if _, err := exec.LookPath("kubectl"); err != nil {
			t.Skip("kubectl not found")
		}

		outDir := filepath.Join(t.TempDir(), "kbuild")

		cmd := exec.Command(binaryPath, "scaffold", "Deployment", "Service", "-o", outDir)
		if err := cmd.Run(); err != nil {
			t.Fatalf("scaffold failed: %v", err)
		}

		kustomize := exec.Command("kubectl", "kustomize", outDir)
		var stdout, stderr bytes.Buffer
		kustomize.Stdout = &stdout
		kustomize.Stderr = &stderr
		if err := kustomize.Run(); err != nil {
			t.Fatalf("kustomize build failed: %v\nstderr: %s", err, stderr.String())
		}
		output := stdout.String()
		if !strings.Contains(output, "Deployment") || !strings.Contains(output, "Service") {
			t.Errorf("kustomize build output missing expected kinds\ngot:\n%s", output)
		}
	})
}

func TestManifestToMigrateRoundTrip(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)

	t.Run("generated manifests pass migration check", func(t *testing.T) {
		for _, resourceType := range []string{"Deployment", "Service", "ConfigMap", "Job"} {
			t.Run(resourceType, func(t *testing.T) {
				yaml := runManifest(t, binaryPath, resourceType)
				file := writeTestManifest(t, strings.ToLower(resourceType)+".yaml", yaml)

				out, err := runMigrate(binaryPath, file)
				if err != nil {
					t.Fatalf("migrate failed on generated %s: %v\noutput: %s", resourceType, err, out)
				}
				if !strings.Contains(out, "OK") {
					t.Errorf("generated %s did not pass migrate check\ngot: %s", resourceType, out)
				}
			})
		}
	})
}

func TestScaffoldToMigrateRoundTrip(t *testing.T) {
	binaryPath := findBinary(t)
	ensureCluster(t)

	t.Run("scaffolded manifests pass migration check", func(t *testing.T) {
		outDir := filepath.Join(t.TempDir(), "roundtrip")

		cmd := exec.Command(binaryPath, "scaffold", "Deployment", "Service", "-o", outDir)
		if err := cmd.Run(); err != nil {
			t.Fatalf("scaffold failed: %v", err)
		}

		for _, file := range []string{"deployment.yaml", "service.yaml"} {
			t.Run(file, func(t *testing.T) {
				path := filepath.Join(outDir, file)
				out, err := runMigrate(binaryPath, path)
				if err != nil {
					t.Fatalf("migrate failed on scaffolded %s: %v\noutput: %s", file, err, out)
				}
				if !strings.Contains(out, "OK") {
					t.Errorf("scaffolded %s did not pass migrate\ngot: %s", file, out)
				}
			})
		}
	})
}

func writeTestManifest(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write test manifest %s: %v", name, err)
	}
	return path
}

func runMigrate(binary, file string) (string, error) {
	cmd := exec.Command(binary, "migrate", file)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String() + stderr.String(), err
}
