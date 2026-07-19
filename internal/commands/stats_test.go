package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestStats(t *testing.T) {
	dir := setupTestProject(t)
	initTestEnv(t, dir)

	// Two rounds of "code changed, doc reviewed with no edit": ack twice with
	// different floors.
	for i, body := range []string{"one", "two"} {
		os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() { // `+body+`
}
`), 0644)
		commitAll(t, dir, "Change handler "+body)

		ackTo = ""
		ackAmend = false
		ackRequired = false
		var ackOut bytes.Buffer
		ackCmd.SetOut(&ackOut)
		if err := ackCmd.RunE(ackCmd, []string{"docs/API.md"}); err != nil {
			t.Fatalf("ack round %d failed: %v", i, err)
		}
		commitAll(t, dir, "Record ack "+body)
	}

	statsJSON = false
	var stdout bytes.Buffer
	statsCmd.SetOut(&stdout)
	if err := statsCmd.RunE(statsCmd, nil); err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	out := stdout.String()

	// docs/API.md: edited once (initial commit), acked twice.
	if !strings.Contains(out, "docs/API.md") {
		t.Fatalf("stats should list docs/API.md, got:\n%s", out)
	}
	var ackCol, editCol int
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "docs/API.md") {
			fields := strings.Fields(line)
			if len(fields) < 4 {
				t.Fatalf("unexpected stats row %q", line)
			}
			ackCol, _ = strconv.Atoi(fields[0])
			editCol, _ = strconv.Atoi(fields[1])
		}
	}
	if ackCol != 2 || editCol != 1 {
		t.Errorf("docs/API.md acks=%d edits=%d, want acks=2 edits=1\n%s", ackCol, editCol, out)
	}
	if !strings.Contains(out, "consider 'gate: structural'") {
		t.Errorf("high ack ratio should carry the structural/deletion hint, got:\n%s", out)
	}

	// GUIDE.md was never acked and edited once: no hint.
	statsJSON = true
	defer func() { statsJSON = false }()
	stdout.Reset()
	if err := statsCmd.RunE(statsCmd, nil); err != nil {
		t.Fatalf("stats --json failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "\"acks\": 2") {
		t.Errorf("JSON should carry ack counts, got:\n%s", stdout.String())
	}
}
