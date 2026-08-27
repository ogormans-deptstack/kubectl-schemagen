package migrate

import (
	"testing"
)

func TestParseManifests(t *testing.T) {
	t.Run("single document with apiVersion and kind", func(t *testing.T) {
		input := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
`
		manifests, err := ParseManifests([]byte(input))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(manifests) != 1 {
			t.Fatalf("expected 1 manifest, got %d", len(manifests))
		}
		if manifests[0].APIVersion != "apps/v1" {
			t.Errorf("expected apiVersion=apps/v1, got %q", manifests[0].APIVersion)
		}
		if manifests[0].Kind != "Deployment" {
			t.Errorf("expected kind=Deployment, got %q", manifests[0].Kind)
		}
	})

	t.Run("multi-document YAML", func(t *testing.T) {
		input := `apiVersion: v1
kind: Service
metadata:
  name: svc
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
`
		manifests, err := ParseManifests([]byte(input))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(manifests) != 2 {
			t.Fatalf("expected 2 manifests, got %d", len(manifests))
		}
		if manifests[0].Kind != "Service" {
			t.Errorf("expected first kind=Service, got %q", manifests[0].Kind)
		}
		if manifests[1].Kind != "Deployment" {
			t.Errorf("expected second kind=Deployment, got %q", manifests[1].Kind)
		}
	})

	t.Run("skips empty documents", func(t *testing.T) {
		input := `---
---
apiVersion: v1
kind: Pod
metadata:
  name: test
---
`
		manifests, err := ParseManifests([]byte(input))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(manifests) != 1 {
			t.Fatalf("expected 1 manifest, got %d", len(manifests))
		}
	})

	t.Run("missing apiVersion returns error in manifest", func(t *testing.T) {
		input := `kind: Deployment
metadata:
  name: web
`
		manifests, err := ParseManifests([]byte(input))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(manifests) != 1 {
			t.Fatalf("expected 1 manifest, got %d", len(manifests))
		}
		if manifests[0].APIVersion != "" {
			t.Errorf("expected empty apiVersion, got %q", manifests[0].APIVersion)
		}
	})

	t.Run("extracts group and version from apiVersion", func(t *testing.T) {
		input := `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: test
`
		manifests, err := ParseManifests([]byte(input))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if manifests[0].Group != "networking.k8s.io" {
			t.Errorf("expected group=networking.k8s.io, got %q", manifests[0].Group)
		}
		if manifests[0].Version != "v1" {
			t.Errorf("expected version=v1, got %q", manifests[0].Version)
		}
	})

	t.Run("core API group has empty group", func(t *testing.T) {
		input := `apiVersion: v1
kind: Pod
metadata:
  name: test
`
		manifests, err := ParseManifests([]byte(input))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if manifests[0].Group != "" {
			t.Errorf("expected empty group for core API, got %q", manifests[0].Group)
		}
		if manifests[0].Version != "v1" {
			t.Errorf("expected version=v1, got %q", manifests[0].Version)
		}
	})
}

func TestCheckDeprecations(t *testing.T) {
	available := map[string][]string{
		"apps":              {"v1"},
		"":                  {"v1"},
		"networking.k8s.io": {"v1"},
		"extensions":        {},
		"policy":            {"v1"},
	}

	t.Run("current API version is not flagged", func(t *testing.T) {
		m := Manifest{APIVersion: "apps/v1", Kind: "Deployment", Group: "apps", Version: "v1"}
		result := CheckAgainstAvailable(m, available)
		if result.Status != StatusOK {
			t.Errorf("expected StatusOK, got %v", result.Status)
		}
	})

	t.Run("removed API group is flagged", func(t *testing.T) {
		m := Manifest{APIVersion: "extensions/v1beta1", Kind: "Deployment", Group: "extensions", Version: "v1beta1"}
		result := CheckAgainstAvailable(m, available)
		if result.Status != StatusRemoved {
			t.Errorf("expected StatusRemoved, got %v", result.Status)
		}
		if result.Replacement != "apps/v1" {
			t.Errorf("expected replacement apps/v1, got %q", result.Replacement)
		}
	})

	t.Run("removed extensions Ingress suggests networking.k8s.io", func(t *testing.T) {
		m := Manifest{APIVersion: "extensions/v1beta1", Kind: "Ingress", Group: "extensions", Version: "v1beta1"}
		result := CheckAgainstAvailable(m, available)
		if result.Status != StatusRemoved {
			t.Errorf("expected StatusRemoved, got %v", result.Status)
		}
		if result.Replacement != "networking.k8s.io/v1" {
			t.Errorf("expected replacement networking.k8s.io/v1, got %q", result.Replacement)
		}
	})

	t.Run("unknown group is flagged as removed", func(t *testing.T) {
		m := Manifest{APIVersion: "custom.io/v1alpha1", Kind: "Foo", Group: "custom.io", Version: "v1alpha1"}
		result := CheckAgainstAvailable(m, available)
		if result.Status != StatusRemoved {
			t.Errorf("expected StatusRemoved, got %v", result.Status)
		}
	})

	t.Run("deprecated version with newer available is flagged", func(t *testing.T) {
		m := Manifest{APIVersion: "policy/v1beta1", Kind: "PodDisruptionBudget", Group: "policy", Version: "v1beta1"}
		result := CheckAgainstAvailable(m, available)
		if result.Status != StatusDeprecated {
			t.Errorf("expected StatusDeprecated, got %v", result.Status)
		}
		if result.Replacement == "" {
			t.Error("expected a replacement suggestion")
		}
	})

	t.Run("core API v1 is ok", func(t *testing.T) {
		m := Manifest{APIVersion: "v1", Kind: "Pod", Group: "", Version: "v1"}
		result := CheckAgainstAvailable(m, available)
		if result.Status != StatusOK {
			t.Errorf("expected StatusOK, got %v", result.Status)
		}
	})
}

func TestAnalyzeFile(t *testing.T) {
	available := map[string][]string{
		"apps": {"v1"},
		"":     {"v1"},
	}

	t.Run("mixed file returns results per document", func(t *testing.T) {
		input := `apiVersion: v1
kind: Service
metadata:
  name: svc
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: web
`
		results, err := AnalyzeBytes([]byte(input), available)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		if results[0].Status != StatusOK {
			t.Errorf("Service should be OK, got %v", results[0].Status)
		}
		if results[1].Status != StatusRemoved {
			t.Errorf("extensions/v1beta1 Deployment should be Removed, got %v", results[1].Status)
		}
		if results[1].Replacement != "apps/v1" {
			t.Errorf("expected replacement apps/v1, got %q", results[1].Replacement)
		}
	})
}
