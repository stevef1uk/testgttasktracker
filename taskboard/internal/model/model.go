package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Length limits, counted in Unicode code points (utf8.RuneCountInString).
const (
	MaxProjectNameLen = 80
	MaxTaskTitleLen   = 120
	MaxDescriptionLen = 1000
)

// Allowed enum values for task status and priority (SPEC §5).
var (
	allowedStatuses   = []string{"todo", "doing", "done"}
	allowedPriorities = []string{"low", "medium", "high"}
)

// Project is a persisted project row (SPEC §5).
type Project struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

// Task is a persisted task row (SPEC §5).
type Task struct {
	ID          int64
	ProjectID   int64
	Title       string
	Description string
	Status      string
	Priority    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ErrEmptyName is returned when a required name or title is blank.
var ErrEmptyName = errors.New("value must not be empty")

// ValidateProjectName checks that a project name is non-blank and within MaxProjectNameLen code points.
func ValidateProjectName(name string) error {
	return validateLength("project name", name, MaxProjectNameLen)
}

// ValidateTaskTitle checks that a task title is non-blank and within MaxTaskTitleLen code points.
func ValidateTaskTitle(title string) error {
	return validateLength("task title", title, MaxTaskTitleLen)
}

// NormalizeProjectName trims surrounding whitespace from a project name.
func NormalizeProjectName(name string) string {
	return strings.TrimSpace(name)
}

// NormalizeTaskTitle trims surrounding whitespace from a task title.
func NormalizeTaskTitle(title string) string {
	return strings.TrimSpace(title)
}

// ValidateDescription checks that a description is within MaxDescriptionLen code points.
// Descriptions are not trimmed; empty descriptions are allowed.
func ValidateDescription(description string) error {
	if n := utf8.RuneCountInString(description); n > MaxDescriptionLen {
		return fmt.Errorf("description: length %d exceeds maximum %d code points", n, MaxDescriptionLen)
	}
	return nil
}

// ValidateStatus accepts only todo, doing, or done.
func ValidateStatus(status string) error {
	return validateEnum("status", status, allowedStatuses)
}

// ValidatePriority accepts only low, medium, or high.
func ValidatePriority(priority string) error {
	return validateEnum("priority", priority, allowedPriorities)
}

func validateEnum(field, value string, allowed []string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return fmt.Errorf("%s: %q is not one of %s", field, value, strings.Join(allowed, ", "))
}

// validateLength trims the value, rejects blanks, and counts code points after trimming.
func validateLength(field, value string, max int) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("%s: %w", field, ErrEmptyName)
	}
	if n := utf8.RuneCountInString(trimmed); n > max {
		return fmt.Errorf("%s: length %d exceeds maximum %d code points", field, n, max)
	}
	return nil
}
