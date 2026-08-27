package cli

import (
	"fmt"
	"strings"
)

// ManifestOptions holds flags shared by manifest-generation subcommands.
type ManifestOptions struct {
	Name        string
	Image       string
	Replicas    int
	ReplicasSet bool
	Set         []string
	Kubeconfig  string
}

// CollectOverrides merges typed flags and --set pairs into a single map.
// --set values are applied after typed flags and can override them.
func CollectOverrides(opts *ManifestOptions) (map[string]string, error) {
	overrides := make(map[string]string)
	if opts.Name != "" {
		overrides["name"] = opts.Name
	}
	if opts.Image != "" {
		overrides["image"] = opts.Image
	}
	if opts.ReplicasSet {
		overrides["replicas"] = fmt.Sprintf("%d", opts.Replicas)
	}
	for _, s := range opts.Set {
		parts := strings.SplitN(s, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --set value %q: expected KEY=VALUE format", s)
		}
		overrides[parts[0]] = parts[1]
	}
	return overrides, nil
}
