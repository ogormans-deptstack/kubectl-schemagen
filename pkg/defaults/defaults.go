package defaults

import (
	"fmt"
	"strings"
)

func ValueForField(fieldName, schemaType, format, kind string) any {
	if v, ok := fieldNameDefaults(fieldName, kind); ok {
		return v
	}
	return typeDefault(schemaType, format)
}

func FieldDefault(fieldName, kind string) (any, bool) {
	return fieldNameDefaults(fieldName, kind)
}

func TypeDefault(schemaType, format string) any {
	return typeDefault(schemaType, format)
}

func fieldNameDefaults(field, kind string) (any, bool) {
	lower := strings.ToLower(field)
	kindLower := strings.ToLower(kind)

	switch lower {
	case "name":
		return fmt.Sprintf("example-%s", kindLower), true
	case "image":
		return "nginx:latest", true
	case "replicas":
		return 3, true
	case "schedule":
		return "*/5 * * * *", true
	case "restartpolicy":
		if kindLower == "job" || kindLower == "cronjob" {
			return "Never", true
		}
		return "Always", true
	case "containerport", "port", "targetport", "number":
		return 80, true
	case "protocol":
		return "TCP", true
	case "type":
		if kindLower == "service" {
			return "ClusterIP", true
		}
		if kindLower == "secret" {
			return "Opaque", true
		}
		if kindLower == "limitrange" {
			return "Container", true
		}
		return nil, false
	case "accessmodes":
		return []string{"ReadWriteOnce"}, true
	case "storage":
		return "1Gi", true
	case "cpu":
		return "250m", true
	case "memory":
		return "128Mi", true
	case "path":
		return "/", true
	case "pathtype":
		return "Prefix", true
	case "host":
		return "example.com", true
	case "servicename":
		return fmt.Sprintf("example-%s", kindLower), true
	case "minreplicas":
		return 1, true
	case "maxreplicas":
		return 10, true
	case "matchlabels":
		return map[string]string{"app.kubernetes.io/name": fmt.Sprintf("example-%s", kindLower)}, true
	case "metrics":
		if kindLower == "horizontalpodautoscaler" {
			return []any{
				map[string]any{
					"type": "Resource",
					"resource": map[string]any{
						"name": "cpu",
						"target": map[string]any{
							"type":               "Utilization",
							"averageUtilization": 80,
						},
					},
				},
			}, true
		}
		return nil, false
	case "scaletargetref":
		if kindLower == "horizontalpodautoscaler" {
			return map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"name":       fmt.Sprintf("example-%s", kindLower),
			}, true
		}
		return nil, false
	case "data":
		if kindLower == "configmap" {
			return map[string]string{"key": "value"}, true
		}
		if kindLower == "secret" {
			return map[string]string{"username": "YWRtaW4=", "password": "cGFzc3dvcmQ="}, true
		}
		return nil, false
	case "minavailable":
		return 1, true
	case "hard":
		return map[string]any{
			"cpu":    "10",
			"memory": "20Gi",
			"pods":   "20",
		}, true
	case "provisioner":
		return "kubernetes.io/no-provisioner", true
	case "volumebindingmode":
		return "WaitForFirstConsumer", true
	case "reclaimpolicy":
		return "Delete", true
	case "value":
		if kindLower == "priorityclass" {
			return 1000000, true
		}
		return nil, false
	case "globaldefault":
		return false, true
	case "preemptionpolicy":
		return "PreemptLowerPriority", true
	case "description":
		if kindLower == "priorityclass" {
			return "Example priority class", true
		}
		return nil, false
	case "handler":
		return "example-handler", true
	case "controller":
		return "example.com/ingress-controller", true
	case "admissionreviewversions":
		return []string{"v1"}, true
	case "sideeffects":
		return "None", true
	case "failurepolicy":
		return "Fail", true
	case "timeoutseconds":
		return 5, true
	case "cabundle":
		return "LS0tLS1CRUdJTi...", true
	case "scope":
		if kindLower == "customresourcedefinition" {
			return "Namespaced", true
		}
		return nil, false
	case "group":
		if kindLower == "customresourcedefinition" {
			return "example.com", true
		}
		return nil, false
	case "apigroups":
		return []string{""}, true
	case "resources":
		if kindLower == "role" || kindLower == "clusterrole" {
			return []string{"pods"}, true
		}
		return nil, false
	case "verbs":
		return []string{"get", "list", "watch"}, true
	case "roleref":
		roleKind := "Role"
		roleName := "example-role"
		if kindLower == "clusterrolebinding" {
			roleKind = "ClusterRole"
			roleName = "example-clusterrole"
		}
		return map[string]any{
			"apiGroup": "rbac.authorization.k8s.io",
			"kind":     roleKind,
			"name":     roleName,
		}, true
	case "subjects":
		return []any{
			map[string]any{
				"kind":      "ServiceAccount",
				"name":      "example-sa",
				"namespace": "default",
			},
		}, true
	case "rules":
		if kindLower == "role" || kindLower == "clusterrole" {
			return []any{
				map[string]any{
					"apiGroups": []string{""},
					"resources": []string{"pods"},
					"verbs":     []string{"get", "list", "watch"},
				},
			}, true
		}
		return nil, false
	case "webhooks":
		return []any{
			map[string]any{
				"name":                    "example-webhook.example.com",
				"admissionReviewVersions": []string{"v1"},
				"sideEffects":             "None",
				"failurePolicy":           "Fail",
				"timeoutSeconds":          5,
				"clientConfig": map[string]any{
					"service": map[string]any{
						"name":      "example-webhook",
						"namespace": "default",
						"path":      "/validate",
					},
				},
				"rules": []any{
					map[string]any{
						"apiGroups":   []string{""},
						"apiVersions": []string{"v1"},
						"operations":  []string{"CREATE", "UPDATE"},
						"resources":   []string{"pods"},
						"scope":       "Namespaced",
					},
				},
			},
		}, true
	case "names":
		if kindLower == "customresourcedefinition" {
			return map[string]any{
				"kind":     "Example",
				"plural":   "examples",
				"singular": "example",
			}, true
		}
		return nil, false
	case "versions":
		if kindLower == "customresourcedefinition" {
			return []any{
				map[string]any{
					"name":    "v1",
					"served":  true,
					"storage": true,
					"schema": map[string]any{
						"openAPIV3Schema": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"spec": map[string]any{
									"type": "object",
								},
							},
						},
					},
				},
			}, true
		}
		return nil, false
	case "capacity":
		return map[string]any{
			"storage": "10Gi",
		}, true
	case "persistentvolumereclaimpolicy":
		return "Retain", true
	case "storageclassname":
		return "standard", true
	case "hostpath":
		return map[string]any{
			"path": "/data",
		}, true
	}
	return nil, false
}

func typeDefault(schemaType, format string) any {
	switch schemaType {
	case "string":
		switch format {
		case "date-time":
			return "2024-01-01T00:00:00Z"
		case "date":
			return "2024-01-01"
		case "email":
			return "user@example.com"
		case "hostname":
			return "example.com"
		case "ipv4", "ip":
			return "192.168.1.1"
		case "ipv6":
			return "::1"
		case "uri", "url":
			return "https://example.com"
		case "uuid":
			return "550e8400-e29b-41d4-a716-446655440000"
		case "byte":
			return "ZXhhbXBsZQ=="
		case "password":
			return "changeme"
		case "duration":
			return "1h0m0s"
		default:
			return "example"
		}
	case "integer":
		return 1
	case "boolean":
		return false
	case "number":
		return 1.0
	}
	return nil
}

var importantFields = map[string]bool{
	"containers":           true,
	"selector":             true,
	"template":             true,
	"ports":                true,
	"rules":                true,
	"schedule":             true,
	"jobtemplate":          true,
	"podselector":          true,
	"ingress":              true,
	"egress":               true,
	"servicename":          true,
	"scaletargetref":       true,
	"metrics":              true,
	"resources":            true,
	"limits":               true,
	"requests":             true,
	"data":                 true,
	"type":                 true,
	"accessmodes":          true,
	"matchlabels":          true,
	"minreplicas":          true,
	"maxreplicas":          true,
	"replicas":             true,
	"volumeclaimtemplates": true,
	"image":                true,
	"http":                 true,
	"paths":                true,
	"pathtype":             true,
	"backend":              true,
	"service":              true,
	"spec":                 true,
	"path":                 true,
	"port":                 true,
	"number":               true,
	"host":                 true,
	"restartpolicy":        true,

	"subjects":          true,
	"roleref":           true,
	"apigroups":         true,
	"verbs":             true,
	"minavailable":      true,
	"hard":              true,
	"provisioner":       true,
	"volumebindingmode": true,
	"reclaimpolicy":     true,
	"globaldefault":     true,
	"value":             true,

	"webhooks":                      true,
	"admissionreviewversions":       true,
	"sideeffects":                   true,
	"clientconfig":                  true,
	"capacity":                      true,
	"persistentvolumereclaimpolicy": true,
	"storageclassname":              true,
	"hostpath":                      true,
	"handler":                       true,
	"controller":                    true,
	"names":                         true,
	"group":                         true,
	"scope":                         true,
	"versions":                      true,
	"description":                   true,
}

func IsImportantField(fieldName string) bool {
	return importantFields[strings.ToLower(fieldName)]
}
