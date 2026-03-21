package validate

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	MaxQuestionLength    = 500
	MaxDescriptionLength = 2000
	MinPasswordLength    = 8
)

func Password(password string) error {
	if len(strings.TrimSpace(password)) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}
	return nil
}

func Question(question string) error {
	q := strings.TrimSpace(question)
	if q == "" {
		return fmt.Errorf("question is required")
	}
	if len(q) > MaxQuestionLength {
		return fmt.Errorf("question must be %d characters or fewer", MaxQuestionLength)
	}
	return nil
}

func Description(description string) error {
	if len(strings.TrimSpace(description)) > MaxDescriptionLength {
		return fmt.Errorf("description must be %d characters or fewer", MaxDescriptionLength)
	}
	return nil
}

func URL(value string) error {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("please enter a valid evidence link")
	}
	return nil
}

func MarketTiming(now, closesAt, resolvesAt time.Time) error {
	if closesAt.Before(now) || closesAt.Equal(now) {
		return fmt.Errorf("closes_at must be in the future")
	}
	if !closesAt.Before(resolvesAt) {
		return fmt.Errorf("closes_at must be before resolves_at")
	}
	return nil
}
