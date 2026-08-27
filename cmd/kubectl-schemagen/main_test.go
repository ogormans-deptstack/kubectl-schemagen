package main

import (
	"testing"
)

func TestRootCommandHasSubcommands(t *testing.T) {
	cmd := newRootCommand()

	subcommands := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subcommands[sub.Name()] = true
	}

	expected := []string{"manifest", "migrate", "scaffold"}
	for _, name := range expected {
		if !subcommands[name] {
			t.Errorf("expected subcommand %q, not found", name)
		}
	}
}

func TestRootCommandVersionSet(t *testing.T) {
	cmd := newRootCommand()
	if cmd.Version != "dev" {
		t.Errorf("expected version=dev, got %q", cmd.Version)
	}
}
