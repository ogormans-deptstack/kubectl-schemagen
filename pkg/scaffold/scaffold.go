package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func WriteKustomizeBase(dir, resourceType string, manifest []byte) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	filename := strings.ToLower(resourceType) + ".yaml"
	resourcePath := filepath.Join(dir, filename)

	if err := os.WriteFile(resourcePath, manifest, 0644); err != nil {
		return fmt.Errorf("write resource file: %w", err)
	}

	resources, err := existingResources(dir)
	if err != nil {
		return err
	}

	if !contains(resources, filename) {
		resources = append(resources, filename)
	}

	kustomization := buildKustomization(resources)
	kPath := filepath.Join(dir, "kustomization.yaml")
	if err := os.WriteFile(kPath, []byte(kustomization), 0644); err != nil {
		return fmt.Errorf("write kustomization.yaml: %w", err)
	}

	return nil
}

func buildKustomization(resources []string) string {
	var b strings.Builder
	b.WriteString("apiVersion: kustomize.config.k8s.io/v1beta1\n")
	b.WriteString("kind: Kustomization\n")
	b.WriteString("resources:\n")
	for _, r := range resources {
		fmt.Fprintf(&b, "- %s\n", r)
	}
	return b.String()
}

func existingResources(dir string) ([]string, error) {
	kPath := filepath.Join(dir, "kustomization.yaml")
	data, err := os.ReadFile(kPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read kustomization.yaml: %w", err)
	}

	var resources []string
	inResources := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "resources:" {
			inResources = true
			continue
		}
		if inResources {
			if strings.HasPrefix(trimmed, "- ") {
				resources = append(resources, strings.TrimPrefix(trimmed, "- "))
			} else {
				break
			}
		}
	}
	return resources, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
