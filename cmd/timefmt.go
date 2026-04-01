package cmd

import "time"

const registryTimeLayout = time.RFC3339

func formatRegistryTime(t time.Time) string {
	return t.UTC().Format(registryTimeLayout)
}

func parseRegistryTime(value string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
	}

	var lastErr error
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
	}

	return time.Time{}, lastErr
}
