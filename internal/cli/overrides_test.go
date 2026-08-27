package cli

import (
	"testing"
)

func mustCollect(t *testing.T, opts *ManifestOptions) map[string]string {
	t.Helper()
	got, err := CollectOverrides(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return got
}

func TestCollectOverrides(t *testing.T) {
	t.Run("name flag sets name override", func(t *testing.T) {
		opts := &ManifestOptions{Name: "my-app"}
		got := mustCollect(t, opts)
		if got["name"] != "my-app" {
			t.Errorf("expected name=my-app, got %q", got["name"])
		}
	})

	t.Run("image flag sets image override", func(t *testing.T) {
		opts := &ManifestOptions{Image: "nginx:1.25"}
		got := mustCollect(t, opts)
		if got["image"] != "nginx:1.25" {
			t.Errorf("expected image=nginx:1.25, got %q", got["image"])
		}
	})

	t.Run("replicas flag sets replicas override", func(t *testing.T) {
		opts := &ManifestOptions{Replicas: 3, ReplicasSet: true}
		got := mustCollect(t, opts)
		if got["replicas"] != "3" {
			t.Errorf("expected replicas=3, got %q", got["replicas"])
		}
	})

	t.Run("replicas=0 is included in overrides", func(t *testing.T) {
		opts := &ManifestOptions{Replicas: 0, ReplicasSet: true}
		got := mustCollect(t, opts)
		if v, ok := got["replicas"]; !ok || v != "0" {
			t.Errorf("expected replicas=0, got %q (present=%v)", v, ok)
		}
	})

	t.Run("replicas default when flag not set is excluded", func(t *testing.T) {
		opts := &ManifestOptions{Replicas: 0, ReplicasSet: false}
		got := mustCollect(t, opts)
		if _, ok := got["replicas"]; ok {
			t.Error("expected replicas to be absent when flag not set")
		}
	})

	t.Run("set flag with key=value adds override", func(t *testing.T) {
		opts := &ManifestOptions{Set: []string{"serviceName=my-svc"}}
		got := mustCollect(t, opts)
		if got["serviceName"] != "my-svc" {
			t.Errorf("expected serviceName=my-svc, got %q", got["serviceName"])
		}
	})

	t.Run("set flag without equals returns error", func(t *testing.T) {
		opts := &ManifestOptions{Set: []string{"badformat"}}
		_, err := CollectOverrides(opts)
		if err == nil {
			t.Error("expected error for --set without equals sign")
		}
	})

	t.Run("set flag with value containing equals splits correctly", func(t *testing.T) {
		opts := &ManifestOptions{Set: []string{"annotation=foo=bar"}}
		got := mustCollect(t, opts)
		if got["annotation"] != "foo=bar" {
			t.Errorf("expected annotation=foo=bar, got %q", got["annotation"])
		}
	})

	t.Run("set overrides typed flags due to ordering", func(t *testing.T) {
		opts := &ManifestOptions{
			Name: "typed-name",
			Set:  []string{"name=set-name"},
		}
		got := mustCollect(t, opts)
		if got["name"] != "set-name" {
			t.Errorf("expected --set to override --name, got %q", got["name"])
		}
	})

	t.Run("empty name flag is excluded", func(t *testing.T) {
		opts := &ManifestOptions{Name: ""}
		got := mustCollect(t, opts)
		if _, ok := got["name"]; ok {
			t.Error("expected empty name to be absent from overrides")
		}
	})

	t.Run("empty image flag is excluded", func(t *testing.T) {
		opts := &ManifestOptions{Image: ""}
		got := mustCollect(t, opts)
		if _, ok := got["image"]; ok {
			t.Error("expected empty image to be absent from overrides")
		}
	})

	t.Run("all flags combined", func(t *testing.T) {
		opts := &ManifestOptions{
			Name:        "web",
			Image:       "nginx:latest",
			Replicas:    5,
			ReplicasSet: true,
			Set:         []string{"serviceName=web-svc", "type=ClusterIP"},
		}
		got := mustCollect(t, opts)
		expected := map[string]string{
			"name":        "web",
			"image":       "nginx:latest",
			"replicas":    "5",
			"serviceName": "web-svc",
			"type":        "ClusterIP",
		}
		for k, v := range expected {
			if got[k] != v {
				t.Errorf("key %q: expected %q, got %q", k, v, got[k])
			}
		}
		if len(got) != len(expected) {
			t.Errorf("expected %d overrides, got %d: %v", len(expected), len(got), got)
		}
	})

	t.Run("multiple set flags accumulate", func(t *testing.T) {
		opts := &ManifestOptions{Set: []string{"a=1", "b=2", "c=3"}}
		got := mustCollect(t, opts)
		if len(got) != 3 {
			t.Errorf("expected 3 overrides, got %d: %v", len(got), got)
		}
	})

	t.Run("set flag with empty value is valid", func(t *testing.T) {
		opts := &ManifestOptions{Set: []string{"key="}}
		got := mustCollect(t, opts)
		if v, ok := got["key"]; !ok || v != "" {
			t.Errorf("expected key with empty value, got %q (present=%v)", v, ok)
		}
	})
}
