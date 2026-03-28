package main

import (
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func TestResolveAgentID_UsesConfiguredID(t *testing.T) {
    dir := t.TempDir()
    id, err := resolveAgentID("configured-agent", filepath.Join(dir, "agent-id"))
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if id != "configured-agent" {
        t.Fatalf("expected configured-agent, got %q", id)
    }
    if _, err := os.Stat(filepath.Join(dir, "agent-id")); !os.IsNotExist(err) {
        t.Fatalf("expected no state file when configured id is used")
    }
}

func TestResolveAgentID_GeneratesAndPersists(t *testing.T) {
    dir := t.TempDir()
    stateFile := filepath.Join(dir, "agent-id")
    id1, err := resolveAgentID("", stateFile)
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if id1 == "" {
        t.Fatalf("expected generated id")
    }
    if strings.Contains(id1, " ") {
        t.Fatalf("expected compact id, got %q", id1)
    }
    data, err := os.ReadFile(stateFile)
    if err != nil {
        t.Fatalf("expected persisted file, got %v", err)
    }
    if strings.TrimSpace(string(data)) != id1 {
        t.Fatalf("expected persisted id %q, got %q", id1, strings.TrimSpace(string(data)))
    }

    id2, err := resolveAgentID("", stateFile)
    if err != nil {
        t.Fatalf("expected no error on reread, got %v", err)
    }
    if id2 != id1 {
        t.Fatalf("expected stable id %q, got %q", id1, id2)
    }
}
