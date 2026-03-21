package forecast

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

func parseRFC3339(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err == nil {
		return parsed, nil
	}
	return time.Parse("2006-01-02T15:04", value)
}

func parseUUIDs(values []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, fmt.Errorf("parse uuid %q: %w", value, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
