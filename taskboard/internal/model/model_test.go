package model

import (
	"strings"
	"testing"
)

func TestValidateProjectName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", true},
		{"blank spaces", "   ", true},
		{"single char", "a", false},
		{"exactly 80 code points", strings.Repeat("a", 80), false},
		{"81 code points rejected", strings.Repeat("a", 81), true},
		{"80 multibyte code points pass", strings.Repeat("é", 80), false},
		{"81 multibyte code points rejected", strings.Repeat("é", 81), true},
		{"trimmed 80 with padding passes", "  " + strings.Repeat("a", 80) + "  ", false},
		{"trimmed 81 with padding rejected", "  " + strings.Repeat("a", 81) + "  ", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProjectName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateProjectName(%d runes) error = %v, wantErr %v", len([]rune(tt.input)), err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeProjectNameTrims(t *testing.T) {
	if got := NormalizeProjectName("  Alpha  "); got != "Alpha" {
		t.Fatalf("NormalizeProjectName = %q, want %q", got, "Alpha")
	}
}

func TestValidateTaskTitle(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", true},
		{"whitespace only", " \t ", true},
		{"exactly 120 code points", strings.Repeat("t", 120), false},
		{"121 code points rejected", strings.Repeat("t", 121), true},
		{"120 multibyte code points pass", strings.Repeat("ü", 120), false},
		{"121 multibyte code points rejected", strings.Repeat("ü", 121), true},
		{"trimmed 120 with padding passes", " " + strings.Repeat("t", 120) + " ", false},
		{"trimmed 121 with padding rejected", " " + strings.Repeat("t", 121) + " ", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTaskTitle(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateTaskTitle(%d runes) error = %v, wantErr %v", len([]rune(tt.input)), err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeTaskTitleTrims(t *testing.T) {
	if got := NormalizeTaskTitle("  Write tests \n"); got != "Write tests" {
		t.Fatalf("NormalizeTaskTitle = %q, want %q", got, "Write tests")
	}
}

func TestValidateDescription(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty allowed", "", false},
		{"exactly 1000 code points", strings.Repeat("d", 1000), false},
		{"1001 code points rejected", strings.Repeat("d", 1001), true},
		{"1000 multibyte code points pass", strings.Repeat("é", 1000), false},
		{"1001 multibyte code points rejected", strings.Repeat("é", 1001), true},
		{"leading spaces are counted not trimmed", strings.Repeat(" ", 1000), false},
		{"1001 spaces rejected", strings.Repeat(" ", 1001), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDescription(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateDescription(%d runes) error = %v, wantErr %v", len([]rune(tt.input)), err, tt.wantErr)
			}
		})
	}
}

func TestValidateStatus(t *testing.T) {
	for _, s := range []string{"todo", "doing", "done"} {
		if err := ValidateStatus(s); err != nil {
			t.Errorf("ValidateStatus(%q) unexpected error: %v", s, err)
		}
	}
	for _, s := range []string{"", "Todo", "DONE", "blocked", " todo"} {
		if err := ValidateStatus(s); err == nil {
			t.Errorf("ValidateStatus(%q) expected error", s)
		}
	}
}

func TestValidatePriority(t *testing.T) {
	for _, p := range []string{"low", "medium", "high"} {
		if err := ValidatePriority(p); err != nil {
			t.Errorf("ValidatePriority(%q) unexpected error: %v", p, err)
		}
	}
	for _, p := range []string{"", "High", "urgent", "critical"} {
		if err := ValidatePriority(p); err == nil {
			t.Errorf("ValidatePriority(%q) expected error", p)
		}
	}
}
