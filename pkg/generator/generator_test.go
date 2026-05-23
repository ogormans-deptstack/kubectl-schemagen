package generator

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ogormans-deptstack/kubectl-schemagen/pkg/openapi"
)

func TestResourceGeneratorInterface(t *testing.T) {
	t.Run("SupportedTypes returns at least 13 core types", func(t *testing.T) {
		gen := newTestGenerator(t)
		types := gen.SupportedTypes()
		if len(types) < 13 {
			t.Errorf("expected at least 13 supported types, got %d: %v", len(types), types)
		}

		coreTypes := []string{
			"Pod", "Deployment", "Service", "ConfigMap", "Secret",
			"Job", "CronJob", "Ingress", "NetworkPolicy",
			"StatefulSet", "DaemonSet", "PersistentVolumeClaim",
			"HorizontalPodAutoscaler",
		}
		supported := make(map[string]bool)
		for _, st := range types {
			supported[st] = true
		}
		for _, ct := range coreTypes {
			if !supported[ct] {
				t.Errorf("core type %s not in SupportedTypes()", ct)
			}
		}
	})
}

func TestGenerateUnknownTypeWithSuggestion(t *testing.T) {
	gen := newTestGenerator(t)

	t.Run("misspelled type includes suggestion", func(t *testing.T) {
		var buf bytes.Buffer
		err := gen.Generate("Deploymnet", nil, &buf)
		if err == nil {
			t.Fatal("expected error for unknown type")
		}
		errMsg := err.Error()
		if !strings.Contains(errMsg, "Did you mean") {
			t.Errorf("expected suggestion in error, got: %s", errMsg)
		}
		if !strings.Contains(errMsg, "Deployment") {
			t.Errorf("expected Deployment in suggestions, got: %s", errMsg)
		}
	})

	t.Run("completely unknown type has no suggestion", func(t *testing.T) {
		var buf bytes.Buffer
		err := gen.Generate("zzzzzzzzzzz", nil, &buf)
		if err == nil {
			t.Fatal("expected error for unknown type")
		}
		errMsg := err.Error()
		if strings.Contains(errMsg, "Did you mean") {
			t.Errorf("expected no suggestion for garbage input, got: %s", errMsg)
		}
	})
}

func TestGenerateYAML(t *testing.T) {
	coreTypes := []struct {
		name             string
		requiredFields   []string
		overrides        map[string]string
		expectedContains []string
	}{
		{
			name:           "Pod",
			requiredFields: []string{"apiVersion: v1", "kind: Pod", "metadata:", "spec:", "containers:"},
		},
		{
			name:             "Deployment",
			requiredFields:   []string{"apiVersion: apps/v1", "kind: Deployment", "spec:", "replicas:", "selector:", "template:"},
			overrides:        map[string]string{"replicas": "5", "name": "web"},
			expectedContains: []string{"replicas: 5", "name: web"},
		},
		{
			name:           "Service",
			requiredFields: []string{"apiVersion: v1", "kind: Service", "spec:", "ports:"},
		},
		{
			name:           "ConfigMap",
			requiredFields: []string{"apiVersion: v1", "kind: ConfigMap", "metadata:"},
		},
		{
			name:           "Secret",
			requiredFields: []string{"apiVersion: v1", "kind: Secret", "metadata:"},
		},
		{
			name:           "Job",
			requiredFields: []string{"apiVersion: batch/v1", "kind: Job", "spec:", "template:"},
		},
		{
			name:           "CronJob",
			requiredFields: []string{"apiVersion: batch/v1", "kind: CronJob", "spec:", "schedule:"},
		},
		{
			name:           "Ingress",
			requiredFields: []string{"apiVersion: networking.k8s.io/v1", "kind: Ingress", "spec:", "rules:"},
		},
		{
			name:           "NetworkPolicy",
			requiredFields: []string{"apiVersion: networking.k8s.io/v1", "kind: NetworkPolicy", "spec:", "podSelector:"},
		},
		{
			name:           "StatefulSet",
			requiredFields: []string{"apiVersion: apps/v1", "kind: StatefulSet", "spec:", "serviceName:"},
		},
		{
			name:           "DaemonSet",
			requiredFields: []string{"apiVersion: apps/v1", "kind: DaemonSet", "spec:", "selector:", "template:"},
		},
		{
			name:           "PersistentVolumeClaim",
			requiredFields: []string{"apiVersion: v1", "kind: PersistentVolumeClaim", "spec:", "accessModes:", "resources:"},
		},
		{
			name:           "HorizontalPodAutoscaler",
			requiredFields: []string{"apiVersion: autoscaling/v2", "kind: HorizontalPodAutoscaler", "spec:", "scaleTargetRef:"},
		},
		{
			name:           "Role",
			requiredFields: []string{"apiVersion: rbac.authorization.k8s.io/v1", "kind: Role", "metadata:", "rules:"},
		},
		{
			name:           "ClusterRole",
			requiredFields: []string{"apiVersion: rbac.authorization.k8s.io/v1", "kind: ClusterRole", "metadata:", "rules:"},
		},
		{
			name:           "RoleBinding",
			requiredFields: []string{"apiVersion: rbac.authorization.k8s.io/v1", "kind: RoleBinding", "metadata:", "subjects:", "roleRef:"},
		},
		{
			name:           "ClusterRoleBinding",
			requiredFields: []string{"apiVersion: rbac.authorization.k8s.io/v1", "kind: ClusterRoleBinding", "metadata:", "subjects:", "roleRef:"},
		},
		{
			name:           "ServiceAccount",
			requiredFields: []string{"apiVersion: v1", "kind: ServiceAccount", "metadata:"},
		},
		{
			name:           "Namespace",
			requiredFields: []string{"apiVersion: v1", "kind: Namespace", "metadata:"},
		},
		{
			name:           "PodDisruptionBudget",
			requiredFields: []string{"apiVersion: policy/v1", "kind: PodDisruptionBudget", "metadata:", "spec:", "selector:"},
		},
		{
			name:           "ResourceQuota",
			requiredFields: []string{"apiVersion: v1", "kind: ResourceQuota", "metadata:", "spec:", "hard:"},
		},
		{
			name:           "LimitRange",
			requiredFields: []string{"apiVersion: v1", "kind: LimitRange", "metadata:", "spec:", "limits:"},
		},
		{
			name:           "PersistentVolume",
			requiredFields: []string{"apiVersion: v1", "kind: PersistentVolume", "metadata:", "spec:", "capacity:", "accessModes:"},
		},
		{
			name:           "IngressClass",
			requiredFields: []string{"apiVersion: networking.k8s.io/v1", "kind: IngressClass", "metadata:", "spec:", "controller:"},
		},
		{
			name:           "StorageClass",
			requiredFields: []string{"apiVersion: storage.k8s.io/v1", "kind: StorageClass", "metadata:", "provisioner:"},
		},
		{
			name:           "PriorityClass",
			requiredFields: []string{"apiVersion: scheduling.k8s.io/v1", "kind: PriorityClass", "metadata:", "value:"},
		},
		{
			name:           "ValidatingWebhookConfiguration",
			requiredFields: []string{"apiVersion: admissionregistration.k8s.io/v1", "kind: ValidatingWebhookConfiguration", "metadata:", "webhooks:"},
		},
		{
			name:           "MutatingWebhookConfiguration",
			requiredFields: []string{"apiVersion: admissionregistration.k8s.io/v1", "kind: MutatingWebhookConfiguration", "metadata:", "webhooks:"},
		},
		{
			name:           "CustomResourceDefinition",
			requiredFields: []string{"apiVersion: apiextensions.k8s.io/v1", "kind: CustomResourceDefinition", "metadata:", "spec:", "group:", "names:", "scope:"},
		},
		{
			name:           "RuntimeClass",
			requiredFields: []string{"apiVersion: node.k8s.io/v1", "kind: RuntimeClass", "metadata:", "handler:"},
		},
	}

	crdTypes := []struct {
		name             string
		requiredFields   []string
		overrides        map[string]string
		expectedContains []string
	}{
		{
			name:             "CronTab",
			requiredFields:   []string{"apiVersion: stable.example.com/v1", "kind: CronTab", "metadata:", "spec:", "cronSpec:", "image:"},
			overrides:        map[string]string{"cronSpec": "*/10 * * * *", "image": "busybox:1.36"},
			expectedContains: []string{"cronSpec: \"*/10 * * * *\"", "image: \"busybox:1.36\""},
		},
		{
			name:           "HTTPRoute",
			requiredFields: []string{"kind: HTTPRoute", "metadata:", "spec:", "parentRefs:", "rules:"},
		},
		{
			name:           "Gateway",
			requiredFields: []string{"kind: Gateway", "metadata:", "spec:", "gatewayClassName:", "listeners:"},
		},
		{
			name:           "GatewayClass",
			requiredFields: []string{"kind: GatewayClass", "metadata:", "spec:", "controllerName:"},
		},
		{
			name:           "GRPCRoute",
			requiredFields: []string{"kind: GRPCRoute", "metadata:", "spec:", "parentRefs:", "rules:"},
		},
		{
			name:           "TCPRoute",
			requiredFields: []string{"kind: TCPRoute", "metadata:", "spec:", "parentRefs:", "rules:"},
		},
		{
			name:           "TLSRoute",
			requiredFields: []string{"kind: TLSRoute", "metadata:", "spec:", "parentRefs:", "rules:"},
		},
		{
			name:           "UDPRoute",
			requiredFields: []string{"kind: UDPRoute", "metadata:", "spec:", "parentRefs:", "rules:"},
		},
		{
			name:           "ReferenceGrant",
			requiredFields: []string{"kind: ReferenceGrant", "metadata:", "spec:", "from:", "to:"},
		},
		{
			name:           "BackendLBPolicy",
			requiredFields: []string{"kind: BackendLBPolicy", "metadata:", "spec:", "targetRefs:"},
		},
		{
			name:           "BackendTLSPolicy",
			requiredFields: []string{"kind: BackendTLSPolicy", "metadata:", "spec:", "targetRefs:", "validation:"},
		},
	}

	gen := newTestGenerator(t)

	allTypes := make([]struct {
		name             string
		requiredFields   []string
		overrides        map[string]string
		expectedContains []string
	}, 0, len(coreTypes)+len(crdTypes))
	allTypes = append(allTypes, coreTypes...)
	allTypes = append(allTypes, crdTypes...)

	for _, tc := range allTypes {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("generates valid YAML with required fields", func(t *testing.T) {
				var buf bytes.Buffer
				overrides := tc.overrides
				if overrides == nil {
					overrides = map[string]string{}
				}
				err := gen.Generate(tc.name, overrides, &buf)
				if err != nil {
					t.Fatalf("Generate(%s) failed: %v", tc.name, err)
				}
				yaml := buf.String()
				for _, field := range tc.requiredFields {
					if !strings.Contains(yaml, field) {
						t.Errorf("YAML for %s missing required field %q\ngot:\n%s", tc.name, field, yaml)
					}
				}
			})

			if tc.overrides != nil {
				t.Run("respects overrides", func(t *testing.T) {
					var buf bytes.Buffer
					err := gen.Generate(tc.name, tc.overrides, &buf)
					if err != nil {
						t.Fatalf("Generate(%s) with overrides failed: %v", tc.name, err)
					}
					yaml := buf.String()
					for _, expected := range tc.expectedContains {
						if !strings.Contains(yaml, expected) {
							t.Errorf("YAML for %s with overrides missing %q\ngot:\n%s", tc.name, expected, yaml)
						}
					}
				})
			}
		})
	}
}

func TestManifestQuality(t *testing.T) {
	gen := newTestGenerator(t)

	generateYAML := func(t *testing.T, kind string) string {
		t.Helper()
		var buf bytes.Buffer
		if err := gen.Generate(kind, map[string]string{}, &buf); err != nil {
			t.Fatalf("Generate(%s) failed: %v", kind, err)
		}
		return buf.String()
	}

	t.Run("Pod has container with image and ports but no command", func(t *testing.T) {
		yaml := generateYAML(t, "Pod")
		assertContains(t, yaml, "image: \"nginx:latest\"")
		assertContains(t, yaml, "containerPort: 80")
		assertNotContains(t, yaml, "command:")
		assertNotContains(t, yaml, "sleep")
	})

	t.Run("Pod has container resources but no pod-level resources", func(t *testing.T) {
		yaml := generateYAML(t, "Pod")
		lines := strings.Split(yaml, "\n")
		containerResourcesSeen := false
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "resources:" {
				if !containerResourcesSeen {
					containerResourcesSeen = true
					continue
				}
				t.Errorf("found second resources: block at line %d (pod-level), should only have container-level\nyaml:\n%s", i+1, yaml)
			}
		}
		if !containerResourcesSeen {
			t.Errorf("expected container-level resources block in Pod\nyaml:\n%s", yaml)
		}
	})

	t.Run("Deployment has RollingUpdate strategy", func(t *testing.T) {
		yaml := generateYAML(t, "Deployment")
		assertContains(t, yaml, "type: RollingUpdate")
		assertNotContains(t, yaml, "type: Recreate")
	})

	t.Run("Deployment has selector matching template labels", func(t *testing.T) {
		yaml := generateYAML(t, "Deployment")
		assertContains(t, yaml, "selector:")
		assertContains(t, yaml, "matchLabels:")
		assertContains(t, yaml, "app.kubernetes.io/name: example-deployment")
	})

	t.Run("name override updates selector matchLabels to match template labels", func(t *testing.T) {
		var buf bytes.Buffer
		if err := gen.Generate("Deployment", map[string]string{"name": "my-app"}, &buf); err != nil {
			t.Fatalf("Generate(Deployment) with name override failed: %v", err)
		}
		yaml := buf.String()
		assertContains(t, yaml, "name: my-app")
		assertContains(t, yaml, "app.kubernetes.io/name: my-app")
		assertNotContains(t, yaml, "app.kubernetes.io/name: example-deployment")
	})

	t.Run("Deployment template has no pod-level resources", func(t *testing.T) {
		yaml := generateYAML(t, "Deployment")
		lines := strings.Split(yaml, "\n")
		inTemplateSpec := false
		for _, line := range lines {
			if strings.Contains(line, "spec:") && inTemplateSpec {
				inTemplateSpec = true
			}
			if strings.HasPrefix(line, "    spec:") {
				inTemplateSpec = true
			}
		}
		resourceCount := strings.Count(yaml, "resources:")
		if resourceCount > 1 {
			t.Errorf("expected only 1 resources block (container-level), found %d\nyaml:\n%s", resourceCount, yaml)
		}
	})

	t.Run("Service has selector and type ClusterIP", func(t *testing.T) {
		yaml := generateYAML(t, "Service")
		assertContains(t, yaml, "type: ClusterIP")
		assertContains(t, yaml, "selector:")
		assertContains(t, yaml, "app.kubernetes.io/name: example-service")
		assertContains(t, yaml, "port: 80")
	})

	t.Run("StatefulSet has RollingUpdate and storage VCT", func(t *testing.T) {
		yaml := generateYAML(t, "StatefulSet")
		assertContains(t, yaml, "updateStrategy:")
		assertContains(t, yaml, "type: RollingUpdate")
		assertNotContains(t, yaml, "type: OnDelete")
		assertContains(t, yaml, "volumeClaimTemplates:")
		assertContains(t, yaml, "storage: \"1Gi\"")
		assertContains(t, yaml, "accessModes:")
		assertContains(t, yaml, "ReadWriteOnce")
		assertContains(t, yaml, "serviceName:")
	})

	t.Run("StatefulSet VCT has no cpu/memory", func(t *testing.T) {
		yaml := generateYAML(t, "StatefulSet")
		vctIdx := strings.Index(yaml, "volumeClaimTemplates:")
		if vctIdx < 0 {
			t.Fatal("StatefulSet missing volumeClaimTemplates")
		}
		vctSection := yaml[vctIdx:]
		if strings.Contains(vctSection, "cpu:") || strings.Contains(vctSection, "memory:") {
			t.Errorf("VCT should have storage, not cpu/memory\nvct section:\n%s", vctSection)
		}
	})

	t.Run("StatefulSet VCT has no selector", func(t *testing.T) {
		yaml := generateYAML(t, "StatefulSet")
		vctIdx := strings.Index(yaml, "volumeClaimTemplates:")
		if vctIdx < 0 {
			t.Fatal("StatefulSet missing volumeClaimTemplates")
		}
		vctSection := yaml[vctIdx:]
		if strings.Contains(vctSection, "selector:") {
			t.Errorf("VCT should not have selector\nvct section:\n%s", vctSection)
		}
	})

	t.Run("DaemonSet has RollingUpdate strategy", func(t *testing.T) {
		yaml := generateYAML(t, "DaemonSet")
		assertContains(t, yaml, "updateStrategy:")
		assertContains(t, yaml, "type: RollingUpdate")
		assertNotContains(t, yaml, "type: OnDelete")
	})

	t.Run("DaemonSet template has no pod-level resources", func(t *testing.T) {
		yaml := generateYAML(t, "DaemonSet")
		resourceCount := strings.Count(yaml, "resources:")
		if resourceCount > 1 {
			t.Errorf("expected only 1 resources block (container-level), found %d\nyaml:\n%s", resourceCount, yaml)
		}
	})

	t.Run("PVC has storage resources and no selector", func(t *testing.T) {
		yaml := generateYAML(t, "PersistentVolumeClaim")
		assertContains(t, yaml, "storage: \"1Gi\"")
		assertContains(t, yaml, "accessModes:")
		assertNotContains(t, yaml, "cpu:")
		assertNotContains(t, yaml, "memory:")
		assertNotContains(t, yaml, "selector:")
	})

	t.Run("Job has restartPolicy Never and template labels", func(t *testing.T) {
		yaml := generateYAML(t, "Job")
		assertContains(t, yaml, "restartPolicy: Never")
		assertContains(t, yaml, "app.kubernetes.io/name: example-job")
		assertNotContains(t, yaml, "selector:")
	})

	t.Run("CronJob has schedule and restartPolicy Never", func(t *testing.T) {
		yaml := generateYAML(t, "CronJob")
		assertContains(t, yaml, "schedule: \"*/5 * * * *\"")
		assertContains(t, yaml, "restartPolicy: Never")
		assertContains(t, yaml, "jobTemplate:")
		assertNotContains(t, yaml, "selector:")
	})

	t.Run("Ingress has host, path, pathType Prefix", func(t *testing.T) {
		yaml := generateYAML(t, "Ingress")
		assertContains(t, yaml, "rules:")
		assertContains(t, yaml, "host: example.com")
		assertContains(t, yaml, "path: /")
		assertContains(t, yaml, "pathType: Prefix")
		assertContains(t, yaml, "number: 80")
	})

	t.Run("HPA has scaleTargetRef and metrics", func(t *testing.T) {
		yaml := generateYAML(t, "HorizontalPodAutoscaler")
		assertContains(t, yaml, "scaleTargetRef:")
		assertContains(t, yaml, "apiVersion: apps/v1")
		assertContains(t, yaml, "kind: Deployment")
		assertContains(t, yaml, "maxReplicas: 10")
		assertContains(t, yaml, "minReplicas: 1")
		assertContains(t, yaml, "metrics:")
		assertContains(t, yaml, "averageUtilization: 80")
	})

	t.Run("ConfigMap has data with key-value pair", func(t *testing.T) {
		yaml := generateYAML(t, "ConfigMap")
		assertContains(t, yaml, "data:")
		assertContains(t, yaml, "key: value")
	})

	t.Run("Secret has data and type Opaque", func(t *testing.T) {
		yaml := generateYAML(t, "Secret")
		assertContains(t, yaml, "data:")
		assertContains(t, yaml, "type: Opaque")
		assertContains(t, yaml, "username:")
		assertContains(t, yaml, "password:")
	})

	t.Run("NetworkPolicy has podSelector and ingress/egress", func(t *testing.T) {
		yaml := generateYAML(t, "NetworkPolicy")
		assertContains(t, yaml, "podSelector:")
		assertContains(t, yaml, "ingress:")
		assertContains(t, yaml, "egress:")
	})

	t.Run("Role has rules with apiGroups, resources, and verbs", func(t *testing.T) {
		yaml := generateYAML(t, "Role")
		assertContains(t, yaml, "rules:")
		assertContains(t, yaml, "apiGroups:")
		assertContains(t, yaml, "resources:")
		assertContains(t, yaml, "verbs:")
	})

	t.Run("ClusterRole has rules with apiGroups, resources, and verbs", func(t *testing.T) {
		yaml := generateYAML(t, "ClusterRole")
		assertContains(t, yaml, "rules:")
		assertContains(t, yaml, "apiGroups:")
		assertContains(t, yaml, "resources:")
		assertContains(t, yaml, "verbs:")
	})

	t.Run("RoleBinding has subjects and roleRef with required fields", func(t *testing.T) {
		yaml := generateYAML(t, "RoleBinding")
		assertContains(t, yaml, "subjects:")
		assertContains(t, yaml, "roleRef:")
		assertContains(t, yaml, "apiGroup:")
		assertContains(t, yaml, "kind:")
		assertContains(t, yaml, "name:")
	})

	t.Run("ClusterRoleBinding has subjects and roleRef", func(t *testing.T) {
		yaml := generateYAML(t, "ClusterRoleBinding")
		assertContains(t, yaml, "subjects:")
		assertContains(t, yaml, "roleRef:")
	})

	t.Run("PodDisruptionBudget has minAvailable and selector", func(t *testing.T) {
		yaml := generateYAML(t, "PodDisruptionBudget")
		assertContains(t, yaml, "selector:")
		assertContains(t, yaml, "minAvailable:")
	})

	t.Run("ResourceQuota has hard with resource limits", func(t *testing.T) {
		yaml := generateYAML(t, "ResourceQuota")
		assertContains(t, yaml, "hard:")
		assertContains(t, yaml, "cpu:")
		assertContains(t, yaml, "memory:")
	})

	t.Run("LimitRange has limits with type Container", func(t *testing.T) {
		yaml := generateYAML(t, "LimitRange")
		assertContains(t, yaml, "limits:")
		assertContains(t, yaml, "type:")
	})

	t.Run("PersistentVolume has capacity and accessModes", func(t *testing.T) {
		yaml := generateYAML(t, "PersistentVolume")
		assertContains(t, yaml, "capacity:")
		assertContains(t, yaml, "storage:")
		assertContains(t, yaml, "accessModes:")
		assertContains(t, yaml, "ReadWriteOnce")
	})

	t.Run("IngressClass has controller field", func(t *testing.T) {
		yaml := generateYAML(t, "IngressClass")
		assertContains(t, yaml, "controller:")
	})

	t.Run("StorageClass has provisioner and volumeBindingMode", func(t *testing.T) {
		yaml := generateYAML(t, "StorageClass")
		assertContains(t, yaml, "provisioner:")
		assertContains(t, yaml, "volumeBindingMode:")
	})

	t.Run("PriorityClass has value and globalDefault", func(t *testing.T) {
		yaml := generateYAML(t, "PriorityClass")
		assertContains(t, yaml, "value:")
		assertContains(t, yaml, "globalDefault:")
	})

	t.Run("ValidatingWebhookConfiguration has webhooks with required fields", func(t *testing.T) {
		yaml := generateYAML(t, "ValidatingWebhookConfiguration")
		assertContains(t, yaml, "webhooks:")
		assertContains(t, yaml, "admissionReviewVersions:")
		assertContains(t, yaml, "sideEffects:")
		assertContains(t, yaml, "clientConfig:")
	})

	t.Run("MutatingWebhookConfiguration has webhooks with required fields", func(t *testing.T) {
		yaml := generateYAML(t, "MutatingWebhookConfiguration")
		assertContains(t, yaml, "webhooks:")
		assertContains(t, yaml, "admissionReviewVersions:")
		assertContains(t, yaml, "sideEffects:")
		assertContains(t, yaml, "clientConfig:")
	})

	t.Run("RuntimeClass has handler field", func(t *testing.T) {
		yaml := generateYAML(t, "RuntimeClass")
		assertContains(t, yaml, "handler:")
	})
}

func TestCRDManifestQuality(t *testing.T) {
	gen := newTestGenerator(t)

	generateYAML := func(t *testing.T, kind string) string {
		t.Helper()
		var buf bytes.Buffer
		if err := gen.Generate(kind, map[string]string{}, &buf); err != nil {
			t.Fatalf("Generate(%s) failed: %v", kind, err)
		}
		return buf.String()
	}

	t.Run("CronTab has required fields cronSpec and image", func(t *testing.T) {
		yaml := generateYAML(t, "CronTab")
		assertContains(t, yaml, "cronSpec:")
		assertContains(t, yaml, "image:")
	})

	t.Run("CronTab has optional fields with defaults", func(t *testing.T) {
		yaml := generateYAML(t, "CronTab")
		assertContains(t, yaml, "replicas:")
		assertContains(t, yaml, "port:")
	})

	t.Run("CronTab restartPolicy enum is present", func(t *testing.T) {
		yaml := generateYAML(t, "CronTab")
		assertContains(t, yaml, "restartPolicy:")
	})

	t.Run("CronTab has correct apiVersion and kind", func(t *testing.T) {
		yaml := generateYAML(t, "CronTab")
		assertContains(t, yaml, "apiVersion: stable.example.com/v1")
		assertContains(t, yaml, "kind: CronTab")
	})

	t.Run("CronTab metadata has name and labels", func(t *testing.T) {
		yaml := generateYAML(t, "CronTab")
		assertContains(t, yaml, "name: example-crontab")
		assertContains(t, yaml, "app.kubernetes.io/name: example-crontab")
	})

	t.Run("CronTab has no status field", func(t *testing.T) {
		yaml := generateYAML(t, "CronTab")
		assertNotContains(t, yaml, "status:")
	})

	t.Run("CronTab image gets nginx default like core types", func(t *testing.T) {
		yaml := generateYAML(t, "CronTab")
		assertContains(t, yaml, "image: \"nginx:latest\"")
	})
}

func TestCRDOverrides(t *testing.T) {
	gen := newTestGenerator(t)

	t.Run("CronTab cronSpec override", func(t *testing.T) {
		var buf bytes.Buffer
		err := gen.Generate("CronTab", map[string]string{"cronSpec": "0 */2 * * *"}, &buf)
		if err != nil {
			t.Fatalf("Generate(CronTab) failed: %v", err)
		}
		yaml := buf.String()
		assertContains(t, yaml, "cronSpec: \"0 */2 * * *\"")
	})

	t.Run("CronTab image override", func(t *testing.T) {
		var buf bytes.Buffer
		err := gen.Generate("CronTab", map[string]string{"image": "redis:7"}, &buf)
		if err != nil {
			t.Fatalf("Generate(CronTab) failed: %v", err)
		}
		yaml := buf.String()
		assertContains(t, yaml, "image: \"redis:7\"")
		assertNotContains(t, yaml, "nginx:latest")
	})

	t.Run("CronTab replicas override", func(t *testing.T) {
		var buf bytes.Buffer
		err := gen.Generate("CronTab", map[string]string{"replicas": "5"}, &buf)
		if err != nil {
			t.Fatalf("Generate(CronTab) failed: %v", err)
		}
		yaml := buf.String()
		assertContains(t, yaml, "replicas: 5")
	})

	t.Run("CronTab name override propagates to labels", func(t *testing.T) {
		var buf bytes.Buffer
		err := gen.Generate("CronTab", map[string]string{"name": "my-cron"}, &buf)
		if err != nil {
			t.Fatalf("Generate(CronTab) failed: %v", err)
		}
		yaml := buf.String()
		assertContains(t, yaml, "name: my-cron")
		assertContains(t, yaml, "app.kubernetes.io/name: my-cron")
	})
}

func TestCRDSingularize(t *testing.T) {
	gen := newTestGenerator(t)

	t.Run("crontab resolves to CronTab", func(t *testing.T) {
		var buf bytes.Buffer
		err := gen.Generate("crontab", map[string]string{}, &buf)
		if err != nil {
			t.Fatalf("Generate(crontab) failed: %v", err)
		}
		assertContains(t, buf.String(), "kind: CronTab")
	})

	t.Run("CronTab resolves case-insensitively", func(t *testing.T) {
		var buf bytes.Buffer
		err := gen.Generate("CRONTAB", map[string]string{}, &buf)
		if err != nil {
			t.Fatalf("Generate(CRONTAB) failed: %v", err)
		}
		assertContains(t, buf.String(), "kind: CronTab")
	})
}

func TestGatewayAPIManifestQuality(t *testing.T) {
	gen := newTestGenerator(t)

	generateYAML := func(t *testing.T, kind string) string {
		t.Helper()
		var buf bytes.Buffer
		if err := gen.Generate(kind, map[string]string{}, &buf); err != nil {
			t.Fatalf("Generate(%s) failed: %v", kind, err)
		}
		return buf.String()
	}

	t.Run("GatewayClass controllerName has domain/path format", func(t *testing.T) {
		yaml := generateYAML(t, "GatewayClass")
		lines := strings.Split(yaml, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "controllerName:") {
				val := strings.TrimPrefix(trimmed, "controllerName: ")
				val = strings.Trim(val, "\"")
				if !strings.Contains(val, "/") {
					t.Errorf("controllerName should be domain/path format, got %q", val)
				}
				return
			}
		}
		t.Fatal("GatewayClass missing controllerName field")
	})

	t.Run("Gateway does not include addresses (optional, complex validation)", func(t *testing.T) {
		yaml := generateYAML(t, "Gateway")
		assertNotContains(t, yaml, "addresses:")
	})

	t.Run("Gateway listeners have required fields", func(t *testing.T) {
		yaml := generateYAML(t, "Gateway")
		assertContains(t, yaml, "listeners:")
		assertContains(t, yaml, "port:")
		assertContains(t, yaml, "protocol:")
	})

	t.Run("HTTPRoute filters include sibling when type is discriminated", func(t *testing.T) {
		yaml := generateYAML(t, "HTTPRoute")
		if strings.Contains(yaml, "type: RequestHeaderModifier") {
			assertContains(t, yaml, "requestHeaderModifier:")
		}
		if strings.Contains(yaml, "type: ResponseHeaderModifier") {
			assertContains(t, yaml, "responseHeaderModifier:")
		}
	})

	t.Run("HTTPRoute has parentRefs with name", func(t *testing.T) {
		yaml := generateYAML(t, "HTTPRoute")
		assertContains(t, yaml, "parentRefs:")
		assertContains(t, yaml, "name:")
	})

	t.Run("HTTPRoute rules have backendRefs", func(t *testing.T) {
		yaml := generateYAML(t, "HTTPRoute")
		assertContains(t, yaml, "backendRefs:")
	})

	t.Run("BackendTLSPolicy subjectAltNames item includes hostname when type is Hostname", func(t *testing.T) {
		yaml := generateYAML(t, "BackendTLSPolicy")
		sanIdx := strings.Index(yaml, "subjectAltNames:")
		if sanIdx < 0 {
			return
		}
		sanSection := yaml[sanIdx:]
		if strings.Contains(sanSection, "type: Hostname") {
			if !strings.Contains(sanSection, "hostname:") {
				t.Errorf("subjectAltNames item with type: Hostname must include hostname field\nsubjectAltNames section:\n%s", sanSection)
			}
		}
	})

	t.Run("TCPRoute passes dry-run structure", func(t *testing.T) {
		yaml := generateYAML(t, "TCPRoute")
		assertContains(t, yaml, "parentRefs:")
		assertContains(t, yaml, "rules:")
		assertContains(t, yaml, "backendRefs:")
	})

	t.Run("ReferenceGrant has from and to", func(t *testing.T) {
		yaml := generateYAML(t, "ReferenceGrant")
		assertContains(t, yaml, "from:")
		assertContains(t, yaml, "to:")
	})
}

func TestSchemaDefaults(t *testing.T) {
	gen := newTestGenerator(t)

	generateYAML := func(t *testing.T, kind string) string {
		t.Helper()
		var buf bytes.Buffer
		if err := gen.Generate(kind, map[string]string{}, &buf); err != nil {
			t.Fatalf("Generate(%s) failed: %v", kind, err)
		}
		return buf.String()
	}

	t.Run("schema default values are used over type defaults", func(t *testing.T) {
		yaml := generateYAML(t, "GatewayClass")
		assertContains(t, yaml, "controllerName:")
		if strings.Contains(yaml, "controllerName: example\n") {
			t.Errorf("controllerName should not be generic 'example' - schema has pattern constraint\nyaml:\n%s", yaml)
		}
	})
}

func TestPatternAwareDefaults(t *testing.T) {
	t.Run("generates domain/path for slash patterns", func(t *testing.T) {
		result := generatePatternExample("^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*\\/[A-Za-z0-9\\/\\-._~%!$&'()*+,;=:]+$")
		if !strings.Contains(result, "/") {
			t.Errorf("pattern with slash should produce domain/path value, got %q", result)
		}
	})

	t.Run("returns empty for non-slash patterns", func(t *testing.T) {
		result := generatePatternExample("^[a-z0-9]+$")
		if result != "" {
			t.Errorf("simple pattern should return empty (let other defaults handle it), got %q", result)
		}
	})
}

func TestExcludedFields(t *testing.T) {
	gen := newTestGenerator(t)

	workloadTypes := []string{"Pod", "Deployment", "Job", "CronJob", "StatefulSet", "DaemonSet"}

	for _, kind := range workloadTypes {
		t.Run(kind+" excludes internal/noise fields", func(t *testing.T) {
			var buf bytes.Buffer
			if err := gen.Generate(kind, map[string]string{}, &buf); err != nil {
				t.Fatalf("Generate(%s) failed: %v", kind, err)
			}
			yaml := buf.String()

			excluded := []string{
				"status:",
				"managedFields:",
				"resourceVersion:",
				"uid:",
				"creationTimestamp:",
				"selfLink:",
				"finalizers:",
				"ownerReferences:",
				"initContainers:",
				"volumes:",
				"volumeMounts:",
				"livenessProbe:",
				"readinessProbe:",
				"startupProbe:",
				"env:",
				"securityContext:",
				"lifecycle:",
				"affinity:",
				"command:",
			}
			for _, field := range excluded {
				if strings.Contains(yaml, field) {
					t.Errorf("%s YAML should not contain %q\nyaml:\n%s", kind, field, yaml)
				}
			}
		})
	}

	t.Run("PVC excludes selector", func(t *testing.T) {
		var buf bytes.Buffer
		if err := gen.Generate("PersistentVolumeClaim", map[string]string{}, &buf); err != nil {
			t.Fatalf("Generate(PVC) failed: %v", err)
		}
		assertNotContains(t, buf.String(), "selector:")
	})

	t.Run("Pod-level resources excluded from workloads with containers", func(t *testing.T) {
		for _, kind := range workloadTypes {
			t.Run(kind, func(t *testing.T) {
				var buf bytes.Buffer
				if err := gen.Generate(kind, map[string]string{}, &buf); err != nil {
					t.Fatalf("Generate(%s) failed: %v", kind, err)
				}
				yaml := buf.String()
				assertNotContains(t, yaml, "claims:")
			})
		}
	})
}

func TestOverridesExpanded(t *testing.T) {
	gen := newTestGenerator(t)

	t.Run("Service name override", func(t *testing.T) {
		var buf bytes.Buffer
		err := gen.Generate("Service", map[string]string{"name": "my-svc"}, &buf)
		if err != nil {
			t.Fatalf("Generate(Service) failed: %v", err)
		}
		yaml := buf.String()
		assertContains(t, yaml, "name: my-svc")
		assertContains(t, yaml, "app.kubernetes.io/name: my-svc")
	})

	t.Run("CronJob image override", func(t *testing.T) {
		var buf bytes.Buffer
		err := gen.Generate("CronJob", map[string]string{"image": "busybox:1.36"}, &buf)
		if err != nil {
			t.Fatalf("Generate(CronJob) failed: %v", err)
		}
		yaml := buf.String()
		assertContains(t, yaml, "image: \"busybox:1.36\"")
		assertNotContains(t, yaml, "nginx:latest")
	})

	t.Run("Ingress name override propagates to labels and backend", func(t *testing.T) {
		var buf bytes.Buffer
		err := gen.Generate("Ingress", map[string]string{"name": "web-ing"}, &buf)
		if err != nil {
			t.Fatalf("Generate(Ingress) failed: %v", err)
		}
		yaml := buf.String()
		assertContains(t, yaml, "name: web-ing")
		assertContains(t, yaml, "app.kubernetes.io/name: web-ing")
	})

	t.Run("StatefulSet replicas override", func(t *testing.T) {
		var buf bytes.Buffer
		err := gen.Generate("StatefulSet", map[string]string{"replicas": "5"}, &buf)
		if err != nil {
			t.Fatalf("Generate(StatefulSet) failed: %v", err)
		}
		yaml := buf.String()
		assertContains(t, yaml, "replicas: 5")
	})

	t.Run("DaemonSet image override", func(t *testing.T) {
		var buf bytes.Buffer
		err := gen.Generate("DaemonSet", map[string]string{"image": "fluentd:v1.16"}, &buf)
		if err != nil {
			t.Fatalf("Generate(DaemonSet) failed: %v", err)
		}
		yaml := buf.String()
		assertContains(t, yaml, "image: \"fluentd:v1.16\"")
		assertNotContains(t, yaml, "nginx:latest")
	})

	t.Run("Job name override", func(t *testing.T) {
		var buf bytes.Buffer
		err := gen.Generate("Job", map[string]string{"name": "batch-job"}, &buf)
		if err != nil {
			t.Fatalf("Generate(Job) failed: %v", err)
		}
		yaml := buf.String()
		assertContains(t, yaml, "name: batch-job")
		assertContains(t, yaml, "app.kubernetes.io/name: batch-job")
	})
}

func assertContains(t *testing.T, yaml, expected string) {
	t.Helper()
	if !strings.Contains(yaml, expected) {
		t.Errorf("YAML missing expected %q\ngot:\n%s", expected, yaml)
	}
}

func assertNotContains(t *testing.T, yaml, forbidden string) {
	t.Helper()
	if strings.Contains(yaml, forbidden) {
		t.Errorf("YAML should NOT contain %q\ngot:\n%s", forbidden, yaml)
	}
}

func TestAliasResolution(t *testing.T) {
	gen := newTestGenerator(t)

	aliases := map[string]string{
		"po":     "Pod",
		"deploy": "Deployment",
		"svc":    "Service",
		"cm":     "ConfigMap",
		"cj":     "CronJob",
		"ing":    "Ingress",
		"netpol": "NetworkPolicy",
		"sts":    "StatefulSet",
		"ds":     "DaemonSet",
		"pvc":    "PersistentVolumeClaim",
		"hpa":    "HorizontalPodAutoscaler",
		"sa":     "ServiceAccount",
		"ns":     "Namespace",
		"pdb":    "PodDisruptionBudget",
		"pv":     "PersistentVolume",
		"sc":     "StorageClass",
		"pc":     "PriorityClass",
		"quota":  "ResourceQuota",
	}

	for alias, expectedKind := range aliases {
		t.Run(alias, func(t *testing.T) {
			var buf bytes.Buffer
			err := gen.Generate(alias, map[string]string{}, &buf)
			if err != nil {
				t.Fatalf("Generate(%s) failed: %v", alias, err)
			}
			if !strings.Contains(buf.String(), "kind: "+expectedKind) {
				t.Errorf("alias %s should generate %s, got:\n%s", alias, expectedKind, buf.String())
			}
		})
	}
}

func TestSupportedTypesWithAliases(t *testing.T) {
	gen := newTestGenerator(t).(*OpenAPIGenerator)

	t.Run("includes aliases in parentheses for types that have them", func(t *testing.T) {
		lines := gen.SupportedTypesWithAliases()
		found := make(map[string]bool)
		for _, line := range lines {
			found[line] = true
		}

		expected := map[string]string{
			"Deployment":              "deploy",
			"Service":                 "svc",
			"ConfigMap":               "cm",
			"StatefulSet":             "sts",
			"DaemonSet":               "ds",
			"PersistentVolumeClaim":   "pvc",
			"CronJob":                 "cj",
			"HorizontalPodAutoscaler": "hpa",
		}

		for kind, alias := range expected {
			matchFound := false
			for _, line := range lines {
				if strings.HasPrefix(line, kind+" (") && strings.Contains(line, alias) {
					matchFound = true
					break
				}
			}
			if !matchFound {
				t.Errorf("expected %s to show alias %q in SupportedTypesWithAliases(), lines: %v", kind, alias, lines)
			}
		}
	})

	t.Run("types without aliases have no parentheses", func(t *testing.T) {
		lines := gen.SupportedTypesWithAliases()
		for _, line := range lines {
			if strings.HasPrefix(line, "CronTab") {
				if strings.Contains(line, "(") {
					t.Errorf("CronTab should have no aliases, got: %s", line)
				}
			}
		}
	})

	t.Run("is sorted alphabetically by kind", func(t *testing.T) {
		lines := gen.SupportedTypesWithAliases()
		for i := 1; i < len(lines); i++ {
			prevKind := strings.SplitN(lines[i-1], " ", 2)[0]
			currKind := strings.SplitN(lines[i], " ", 2)[0]
			if prevKind > currKind {
				t.Errorf("not sorted: %q came before %q", prevKind, currKind)
			}
		}
	})

	t.Run("has same count as SupportedTypes", func(t *testing.T) {
		plain := gen.SupportedTypes()
		withAliases := gen.SupportedTypesWithAliases()
		if len(plain) != len(withAliases) {
			t.Errorf("SupportedTypes has %d entries but SupportedTypesWithAliases has %d", len(plain), len(withAliases))
		}
	})
}

func TestOverrideEdgeCases(t *testing.T) {
	gen := newTestGenerator(t)

	genYAML := func(t *testing.T, kind string, overrides map[string]string) string {
		t.Helper()
		var buf bytes.Buffer
		if err := gen.Generate(kind, overrides, &buf); err != nil {
			t.Fatalf("Generate(%s) failed: %v", kind, err)
		}
		return buf.String()
	}

	t.Run("image override propagates to container in each workload type", func(t *testing.T) {
		kinds := []string{"Deployment", "Pod", "Job", "CronJob", "StatefulSet", "DaemonSet"}
		for _, kind := range kinds {
			t.Run(kind, func(t *testing.T) {
				yaml := genYAML(t, kind, map[string]string{"image": "redis:7-alpine"})
				assertContains(t, yaml, "image: \"redis:7-alpine\"")
				assertNotContains(t, yaml, "nginx:latest")
			})
		}
	})

	t.Run("replicas override works for Deployment and StatefulSet", func(t *testing.T) {
		for _, kind := range []string{"Deployment", "StatefulSet"} {
			t.Run(kind+" replicas=1", func(t *testing.T) {
				yaml := genYAML(t, kind, map[string]string{"replicas": "1"})
				assertContains(t, yaml, "replicas: 1")
			})
			t.Run(kind+" replicas=3", func(t *testing.T) {
				yaml := genYAML(t, kind, map[string]string{"replicas": "3"})
				assertContains(t, yaml, "replicas: 3")
			})
		}
	})

	t.Run("replicas=0 produces replicas: 0", func(t *testing.T) {
		yaml := genYAML(t, "Deployment", map[string]string{"replicas": "0"})
		assertContains(t, yaml, "replicas: 0")
	})

	t.Run("replicas with non-numeric string falls through as raw string", func(t *testing.T) {
		yaml := genYAML(t, "Deployment", map[string]string{"replicas": "notanumber"})
		assertContains(t, yaml, "replicas: notanumber")
	})

	t.Run("replicas with integer string parses as int not quoted string", func(t *testing.T) {
		yaml := genYAML(t, "Deployment", map[string]string{"replicas": "5"})
		assertContains(t, yaml, "replicas: 5")
		assertNotContains(t, yaml, "replicas: \"5\"")
	})

	t.Run("unknown override key is silently ignored", func(t *testing.T) {
		withOverride := genYAML(t, "Deployment", map[string]string{"nonexistent": "value"})
		withoutOverride := genYAML(t, "Deployment", map[string]string{})
		assertNotContains(t, withOverride, "nonexistent")
		if withOverride != withoutOverride {
			t.Errorf("unknown override should not change output\nwith override:\n%s\nwithout:\n%s", withOverride, withoutOverride)
		}
	})

	t.Run("empty override map produces same output as nil overrides", func(t *testing.T) {
		yamlEmpty := genYAML(t, "Deployment", map[string]string{})
		var buf bytes.Buffer
		if err := gen.Generate("Deployment", nil, &buf); err != nil {
			t.Fatalf("Generate with nil overrides failed: %v", err)
		}
		yamlNil := buf.String()
		if yamlEmpty != yamlNil {
			t.Errorf("empty map and nil overrides should produce identical output\nempty:\n%s\nnil:\n%s", yamlEmpty, yamlNil)
		}
	})

	t.Run("empty string name override produces empty name", func(t *testing.T) {
		yaml := genYAML(t, "Deployment", map[string]string{"name": ""})
		assertContains(t, yaml, "name: \"\"")
	})

	t.Run("name with special characters works fine", func(t *testing.T) {
		yaml := genYAML(t, "Deployment", map[string]string{"name": "my-app-v2.0"})
		assertContains(t, yaml, "name: my-app-v2.0")
		assertContains(t, yaml, "app.kubernetes.io/name: my-app-v2.0")
	})

	t.Run("CronJob image override reaches nested container in jobTemplate", func(t *testing.T) {
		yaml := genYAML(t, "CronJob", map[string]string{"image": "curl:8.5"})
		assertContains(t, yaml, "image: \"curl:8.5\"")
		assertNotContains(t, yaml, "nginx:latest")
		assertContains(t, yaml, "jobTemplate:")
	})

	t.Run("set-style override for StatefulSet serviceName applies via schema walk", func(t *testing.T) {
		yaml := genYAML(t, "StatefulSet", map[string]string{"serviceName": "my-svc"})
		assertContains(t, yaml, "serviceName: my-svc")
	})
}

func TestOverridePriority(t *testing.T) {
	gen := newTestGenerator(t)

	genYAML := func(t *testing.T, kind string, overrides map[string]string) string {
		t.Helper()
		var buf bytes.Buffer
		if err := gen.Generate(kind, overrides, &buf); err != nil {
			t.Fatalf("Generate(%s) failed: %v", kind, err)
		}
		return buf.String()
	}

	t.Run("multiple overrides combined: name, image, replicas", func(t *testing.T) {
		yaml := genYAML(t, "Deployment", map[string]string{
			"name":     "web",
			"image":    "nginx:latest",
			"replicas": "3",
		})
		assertContains(t, yaml, "name: web")
		assertContains(t, yaml, "app.kubernetes.io/name: web")
		assertContains(t, yaml, "image: \"nginx:latest\"")
		assertContains(t, yaml, "replicas: 3")
		assertNotContains(t, yaml, "example-deployment")
	})

	t.Run("name override affects metadata and template labels and selector", func(t *testing.T) {
		yaml := genYAML(t, "Deployment", map[string]string{"name": "frontend"})
		assertContains(t, yaml, "name: frontend")
		assertContains(t, yaml, "app.kubernetes.io/name: frontend")
		assertContains(t, yaml, "matchLabels:")
		assertNotContains(t, yaml, "example-deployment")
	})

	t.Run("image override replaces default nginx for Deployment", func(t *testing.T) {
		yaml := genYAML(t, "Deployment", map[string]string{"image": "python:3.12-slim"})
		assertContains(t, yaml, "image: \"python:3.12-slim\"")
		assertNotContains(t, yaml, "nginx:latest")
	})

	t.Run("name and image combined in StatefulSet", func(t *testing.T) {
		yaml := genYAML(t, "StatefulSet", map[string]string{
			"name":  "db",
			"image": "postgres:16",
		})
		assertContains(t, yaml, "name: db")
		assertContains(t, yaml, "app.kubernetes.io/name: db")
		assertContains(t, yaml, "image: \"postgres:16\"")
	})

	t.Run("name and image combined in DaemonSet", func(t *testing.T) {
		yaml := genYAML(t, "DaemonSet", map[string]string{
			"name":  "log-collector",
			"image": "fluentd:v1.17",
		})
		assertContains(t, yaml, "name: log-collector")
		assertContains(t, yaml, "app.kubernetes.io/name: log-collector")
		assertContains(t, yaml, "image: \"fluentd:v1.17\"")
		assertNotContains(t, yaml, "example-daemonset")
	})

	t.Run("overrides with unknown keys mixed in are partially applied", func(t *testing.T) {
		yaml := genYAML(t, "Deployment", map[string]string{
			"name":        "partial",
			"nonexistent": "ignored",
			"replicas":    "2",
		})
		assertContains(t, yaml, "name: partial")
		assertContains(t, yaml, "replicas: 2")
		assertNotContains(t, yaml, "nonexistent")
		assertNotContains(t, yaml, "ignored")
	})

	t.Run("CronJob combined name and image override", func(t *testing.T) {
		yaml := genYAML(t, "CronJob", map[string]string{
			"name":  "nightly-backup",
			"image": "aws-cli:2.15",
		})
		assertContains(t, yaml, "name: nightly-backup")
		assertContains(t, yaml, "app.kubernetes.io/name: nightly-backup")
		assertContains(t, yaml, "image: \"aws-cli:2.15\"")
		assertNotContains(t, yaml, "example-cronjob")
	})
}

func TestDotPathOverrides(t *testing.T) {
	gen := newTestGenerator(t)

	genYAML := func(t *testing.T, kind string, overrides map[string]string) string {
		t.Helper()
		var buf bytes.Buffer
		if err := gen.Generate(kind, overrides, &buf); err != nil {
			t.Fatalf("Generate(%s) failed: %v", kind, err)
		}
		return buf.String()
	}

	t.Run("spec.restartPolicy override on Pod", func(t *testing.T) {
		yaml := genYAML(t, "Pod", map[string]string{"spec.restartPolicy": "Never"})
		assertContains(t, yaml, "restartPolicy: Never")
	})

	t.Run("spec.template.spec.restartPolicy override on Job wins over post-processor", func(t *testing.T) {
		yaml := genYAML(t, "Job", map[string]string{"spec.template.spec.restartPolicy": "OnFailure"})
		assertContains(t, yaml, "restartPolicy: OnFailure")
		assertNotContains(t, yaml, "restartPolicy: Never")
	})

	t.Run("spec.replicas dot-path form works", func(t *testing.T) {
		yaml := genYAML(t, "Deployment", map[string]string{"spec.replicas": "7"})
		assertContains(t, yaml, "replicas: 7")
	})

	t.Run("spec.template.spec.containers[0].image dot-path override", func(t *testing.T) {
		yaml := genYAML(t, "Deployment", map[string]string{
			"spec.template.spec.containers[0].image": "custom:v3",
		})
		assertContains(t, yaml, `image: "custom:v3"`)
	})

	t.Run("dot-path overrides win over post-processors", func(t *testing.T) {
		yaml := genYAML(t, "Deployment", map[string]string{
			"spec.strategy.type": "Recreate",
		})
		assertContains(t, yaml, "type: Recreate")
	})
}

// TestSingleDocProducesSameOutput verifies that generating from a single
// group-version document produces identical output to generating from the
// full merged document.
func TestSingleDocProducesSameOutput(t *testing.T) {
	mergedDoc := loadMergedFixture(t)
	if mergedDoc == nil {
		t.Skip("fixtures not available")
	}

	tests := []struct {
		resourceType string
		fixture      string
	}{
		{"Deployment", "apis_apps_v1.json"},
		{"Pod", "api_v1.json"},
		{"Service", "api_v1.json"},
		{"CronJob", "apis_batch_v1.json"},
		{"Ingress", "apis_networking_v1.json"},
		{"HorizontalPodAutoscaler", "apis_autoscaling_v2.json"},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			fixtureDir := filepath.Join("..", "..", "test", "fixtures", "openapi")
			data, err := os.ReadFile(filepath.Join(fixtureDir, tt.fixture))
			if err != nil {
				t.Skipf("fixture %s not found: %v", tt.fixture, err)
			}
			singleDoc, err := openapi.ParseDocument(data)
			if err != nil {
				t.Fatalf("parse fixture: %v", err)
			}

			mergedGen := NewOpenAPIGenerator(mergedDoc)
			singleGen := NewOpenAPIGenerator(singleDoc)

			var mergedBuf, singleBuf bytes.Buffer
			if err := mergedGen.Generate(tt.resourceType, nil, &mergedBuf); err != nil {
				t.Fatalf("merged generate: %v", err)
			}
			if err := singleGen.Generate(tt.resourceType, nil, &singleBuf); err != nil {
				t.Fatalf("single generate: %v", err)
			}

			if mergedBuf.String() != singleBuf.String() {
				t.Errorf("output differs for %s\n--- merged ---\n%s\n--- single ---\n%s",
					tt.resourceType, mergedBuf.String(), singleBuf.String())
			}
		})
	}
}

func newTestGenerator(t *testing.T) ResourceGenerator {
	t.Helper()
	doc := loadMergedFixture(t)
	return NewOpenAPIGenerator(doc)
}

// BenchmarkGenerateFromMergedDoc measures generation when all schemas are
// merged into one document (the old FetchAll approach).
func BenchmarkGenerateFromMergedDoc(b *testing.B) {
	doc := loadMergedFixtureB(b)
	if doc == nil {
		b.Skip("fixtures not available")
	}

	b.ResetTimer()
	for b.Loop() {
		gen := NewOpenAPIGenerator(doc)
		var buf bytes.Buffer
		if err := gen.Generate("Deployment", nil, &buf); err != nil {
			b.Fatalf("generate: %v", err)
		}
	}
}

// BenchmarkGenerateFromSingleDoc measures generation when only the target
// group-version schema is loaded (the new FetchForResource approach).
func BenchmarkGenerateFromSingleDoc(b *testing.B) {
	doc := loadSingleFixtureB(b, "apis_apps_v1.json")
	if doc == nil {
		b.Skip("fixtures not available")
	}

	b.ResetTimer()
	for b.Loop() {
		gen := NewOpenAPIGenerator(doc)
		var buf bytes.Buffer
		if err := gen.Generate("Deployment", nil, &buf); err != nil {
			b.Fatalf("generate: %v", err)
		}
	}
}

// BenchmarkBuildGVKIndexMerged measures the cost of building the GVK index
// from the full merged document.
func BenchmarkBuildGVKIndexMerged(b *testing.B) {
	doc := loadMergedFixtureB(b)
	if doc == nil {
		b.Skip("fixtures not available")
	}

	b.ResetTimer()
	for b.Loop() {
		buildGVKIndex(doc)
	}
}

// BenchmarkBuildGVKIndexSingle measures the cost of building the GVK index
// from a single group-version document.
func BenchmarkBuildGVKIndexSingle(b *testing.B) {
	doc := loadSingleFixtureB(b, "apis_apps_v1.json")
	if doc == nil {
		b.Skip("fixtures not available")
	}

	b.ResetTimer()
	for b.Loop() {
		buildGVKIndex(doc)
	}
}

// BenchmarkLoadMergedFixture measures the cost of loading and parsing all
// fixture files (simulating FetchAll).
func BenchmarkLoadMergedFixture(b *testing.B) {
	fixtureDir := filepath.Join("..", "..", "test", "fixtures", "openapi")
	if _, err := os.ReadDir(fixtureDir); err != nil {
		b.Skip("fixtures not available")
	}

	b.ResetTimer()
	for b.Loop() {
		doc := loadMergedFixtureB(b)
		if doc == nil {
			b.Fatal("failed to load fixtures")
		}
	}
}

// BenchmarkLoadSingleFixture measures the cost of loading a single fixture
// file (simulating FetchForResource).
func BenchmarkLoadSingleFixture(b *testing.B) {
	fixtureDir := filepath.Join("..", "..", "test", "fixtures", "openapi")
	if _, err := os.ReadDir(fixtureDir); err != nil {
		b.Skip("fixtures not available")
	}

	b.ResetTimer()
	for b.Loop() {
		doc := loadSingleFixtureB(b, "apis_apps_v1.json")
		if doc == nil {
			b.Fatal("failed to load fixture")
		}
	}
}

func loadMergedFixtureB(b *testing.B) *openapi.Document {
	b.Helper()
	fixtureDir := filepath.Join("..", "..", "test", "fixtures", "openapi")
	files, err := os.ReadDir(fixtureDir)
	if err != nil {
		return nil
	}

	merged := map[string]any{
		"components": map[string]any{
			"schemas": map[string]any{},
		},
	}
	mergedSchemas := merged["components"].(map[string]any)["schemas"].(map[string]any)

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(fixtureDir, f.Name()))
		if err != nil {
			b.Fatalf("read fixture %s: %v", f.Name(), err)
		}
		var doc map[string]any
		if err := json.Unmarshal(data, &doc); err != nil {
			b.Fatalf("parse fixture %s: %v", f.Name(), err)
		}
		components, _ := doc["components"].(map[string]any)
		if components == nil {
			continue
		}
		schemas, _ := components["schemas"].(map[string]any)
		for k, v := range schemas {
			mergedSchemas[k] = v
		}
	}

	data, err := json.Marshal(merged)
	if err != nil {
		b.Fatalf("marshal merged doc: %v", err)
	}
	doc, err := openapi.ParseDocument(data)
	if err != nil {
		b.Fatalf("parse merged doc: %v", err)
	}
	return doc
}

func loadSingleFixtureB(b *testing.B, filename string) *openapi.Document {
	b.Helper()
	fixtureDir := filepath.Join("..", "..", "test", "fixtures", "openapi")
	data, err := os.ReadFile(filepath.Join(fixtureDir, filename))
	if err != nil {
		return nil
	}
	doc, err := openapi.ParseDocument(data)
	if err != nil {
		b.Fatalf("parse fixture %s: %v", filename, err)
	}
	return doc
}

func loadMergedFixture(t *testing.T) *openapi.Document {
	t.Helper()
	fixtureDir := filepath.Join("..", "..", "test", "fixtures", "openapi")
	files, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Skipf("OpenAPI fixtures not found at %s: %v (run tests with a kind cluster first)", fixtureDir, err)
		return nil
	}

	merged := map[string]any{
		"components": map[string]any{
			"schemas": map[string]any{},
		},
	}
	mergedSchemas := merged["components"].(map[string]any)["schemas"].(map[string]any)

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(fixtureDir, f.Name()))
		if err != nil {
			t.Fatalf("read fixture %s: %v", f.Name(), err)
		}
		var doc map[string]any
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatalf("parse fixture %s: %v", f.Name(), err)
		}
		components, _ := doc["components"].(map[string]any)
		if components == nil {
			continue
		}
		schemas, _ := components["schemas"].(map[string]any)
		for k, v := range schemas {
			mergedSchemas[k] = v
		}
	}

	data, err := json.Marshal(merged)
	if err != nil {
		t.Fatalf("marshal merged doc: %v", err)
	}
	doc, err := openapi.ParseDocument(data)
	if err != nil {
		t.Fatalf("parse merged doc: %v", err)
	}
	return doc
}
