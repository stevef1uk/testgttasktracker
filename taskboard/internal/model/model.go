package model

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// ErrValidation is returned when client input fails domain validation.
var ErrValidation = errors.New("validation failed")

const (
	maxProjectNameRunes = 80
	maxTitleRunes       = 120
	maxDescriptionRunes = 1000
)

// Project is a named container for tasks.
type Project struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// Task is a Kanban card belonging to one project.
type Task struct {
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"project_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// TaskPatch holds optional task fields. A nil pointer means the field is unchanged.
type TaskPatch struct {
	Title       *string
	Description *string
	Status      *string
	Priority    *string
}

// IsEmpty reports whether the patch changes no fields.
func (p TaskPatch) IsEmpty() bool {
	return p.Title == nil && p.Description == nil && p.Status == nil && p.Priority == nil
}

// ValidateProjectName trims the name and enforces 1-80 code points.
func ValidateProjectName(name string) (string, error) {
	return validateRequiredText("project name", name, maxProjectNameRunes)
}

// ValidateTaskTitle trims the title and enforces 1-120 code points.
func ValidateTaskTitle(title string) (string, error) {
	return validateRequiredText("task title", title, maxTitleRunes)
}

// ValidateDescription enforces at most 1000 code points. Empty is allowed.
func ValidateDescription(description string) (string, error) {
	if utf8.RuneCountInString(description) > maxDescriptionRunes {
		return "", fmt.Errorf("%w: description must be at most %d characters", ErrValidation, maxDescriptionRunes)
	}
	return description, nil
}

// ValidateStatus accepts only todo, doing, or done.
func ValidateStatus(status string) error {
	switch status {
	case "todo", "doing", "done":
		return nil
	}
	return fmt.Errorf("%w: status must be todo, doing, or done", ErrValidation)
}

// ValidatePriority accepts only low, medium, or high.
func ValidatePriority(priority string) error {
	switch priority {
	case "low", "medium", "high":
		return nil
	}
	return fmt.Errorf("%w: priority must be low, medium, or high", ErrValidation)
}

func validateRequiredText(label, value string, maxRunes int) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%w: %s is required", ErrValidation, label)
	}
	if utf8.RuneCountInString(trimmed) > maxRunes {
		return "", fmt.Errorf("%w: %s must be at most %d characters", ErrValidation, label, maxRunes)
	}
	return trimmed, nil
}
