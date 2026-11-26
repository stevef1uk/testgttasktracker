package model

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateProjectName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"trims whitespace", "  Demo  ", "Demo", false},
		{"empty after trim", "   ", "", true},
		{"exactly 80 runes", strings.Repeat("a", 80), strings.Repeat("a", 80), false},
		{"81 runes rejected", strings.Repeat("a", 81), "", true},
		{"80 multibyte runes accepted", strings.Repeat("é", 80), strings.Repeat("é", 80), false},
		{"81 multibyte runes rejected", strings.Repeat("é", 81), "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateProjectName(tt.input)
			if tt.wantErr {
				if !errors.Is(err, ErrValidation) {
					t.Fatalf("error = %v, want ErrValidation", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateTaskTitle(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "Build API", false},
		{"empty", "", true},
		{"whitespace only", " \t ", true},
		{"120 runes accepted", strings.Repeat("t", 120), false},
		{"121 runes rejected", strings.Repeat("t", 121), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateTaskTitle(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateDescription(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty allowed", "", false},
		{"1000 runes accepted", strings.Repeat("d", 1000), false},
		{"1001 runes rejected", strings.Repeat("d", 1001), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateDescription(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateStatusAndPriority(t *testing.T) {
	for _, s := range []string{"todo", "doing", "done"} {
		if err := ValidateStatus(s); err != nil {
			t.Errorf("ValidateStatus(%q) = %v, want nil", s, err)
		}
	}
	for _, s := range []string{"", "TODO", "archived"} {
		if err := ValidateStatus(s); !errors.Is(err, ErrValidation) {
			t.Errorf("ValidateStatus(%q) = %v, want ErrValidation", s, err)
		}
	}
	for _, p := range []string{"low", "medium", "high"} {
		if err := ValidatePriority(p); err != nil {
			t.Errorf("ValidatePriority(%q) = %v, want nil", p, err)
		}
	}
	for _, p := range []string{"", "urgent"} {
		if err := ValidatePriority(p); !errors.Is(err, ErrValidation) {
			t.Errorf("ValidatePriority(%q) = %v, want ErrValidation", p, err)
		}
	}
}

func TestTaskPatchIsEmpty(t *testing.T) {
	empty := TaskPatch{}
	if !empty.IsEmpty() {
		t.Fatal("zero TaskPatch should be empty")
	}
	s := "done"
	if (TaskPatch{Status: &s}).IsEmpty() {
		t.Fatal("TaskPatch with Status set should not be empty")
	}
}
