package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteKustomizeBase(t *testing.T) {
	t.Run("creates kustomization.yaml and resource file", func(t *testing.T) {
		dir := t.TempDir()
		manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 1
`
		err := WriteKustomizeBase(dir, "deployment", []byte(manifest))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		kustomizationPath := filepath.Join(dir, "kustomization.yaml")
		if _, err := os.Stat(kustomizationPath); os.IsNotExist(err) {
			t.Fatal("kustomization.yaml not created")
		}
		kData, err := os.ReadFile(kustomizationPath)
		if err != nil {
			t.Fatalf("reading kustomization.yaml: %v", err)
		}
		kContent := string(kData)
		if !strings.Contains(kContent, "resources:") {
			t.Error("kustomization.yaml missing resources key")
		}
		if !strings.Contains(kContent, "deployment.yaml") {
			t.Error("kustomization.yaml missing deployment.yaml reference")
		}

		resourcePath := filepath.Join(dir, "deployment.yaml")
		if _, err := os.Stat(resourcePath); os.IsNotExist(err) {
			t.Fatal("deployment.yaml not created")
		}
		rData, err := os.ReadFile(resourcePath)
		if err != nil {
			t.Fatalf("reading deployment.yaml: %v", err)
		}
		if string(rData) != manifest {
			t.Errorf("resource content mismatch:\ngot:\n%s\nwant:\n%s", string(rData), manifest)
		}
	})

	t.Run("resource name is lowercased", func(t *testing.T) {
		dir := t.TempDir()
		manifest := `apiVersion: v1
kind: Service
metadata:
  name: web
`
		err := WriteKustomizeBase(dir, "Service", []byte(manifest))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(filepath.Join(dir, "service.yaml")); os.IsNotExist(err) {
			t.Fatal("expected lowercase service.yaml")
		}
	})

	t.Run("creates output directory if missing", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "nested", "base")
		manifest := []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n")

		err := WriteKustomizeBase(dir, "configmap", manifest)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(filepath.Join(dir, "configmap.yaml")); os.IsNotExist(err) {
			t.Fatal("configmap.yaml not created in new directory")
		}
		if _, err := os.Stat(filepath.Join(dir, "kustomization.yaml")); os.IsNotExist(err) {
			t.Fatal("kustomization.yaml not created in new directory")
		}
	})

	t.Run("multiple resources get separate files", func(t *testing.T) {
		dir := t.TempDir()
		deploy := []byte("apiVersion: apps/v1\nkind: Deployment\n")
		svc := []byte("apiVersion: v1\nkind: Service\n")

		if err := WriteKustomizeBase(dir, "deployment", deploy); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := WriteKustomizeBase(dir, "service", svc); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		kData, _ := os.ReadFile(filepath.Join(dir, "kustomization.yaml"))
		kContent := string(kData)
		if !strings.Contains(kContent, "deployment.yaml") {
			t.Error("kustomization.yaml missing deployment.yaml")
		}
		if !strings.Contains(kContent, "service.yaml") {
			t.Error("kustomization.yaml missing service.yaml")
		}
	})
}

func TestKustomizationContent(t *testing.T) {
	t.Run("has apiVersion and kind", func(t *testing.T) {
		content := buildKustomization([]string{"deployment.yaml"})
		if !strings.Contains(content, "apiVersion: kustomize.config.k8s.io/v1beta1") {
			t.Error("missing apiVersion")
		}
		if !strings.Contains(content, "kind: Kustomization") {
			t.Error("missing kind")
		}
	})

	t.Run("lists all resources", func(t *testing.T) {
		content := buildKustomization([]string{"deployment.yaml", "service.yaml"})
		if !strings.Contains(content, "- deployment.yaml") {
			t.Error("missing deployment.yaml in resources")
		}
		if !strings.Contains(content, "- service.yaml") {
			t.Error("missing service.yaml in resources")
		}
	})
}
