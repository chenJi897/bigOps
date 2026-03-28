package service

import (
	"encoding/json"
	"strings"
)

func parseNotifyChannels(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var channels []string
	if err := json.Unmarshal([]byte(raw), &channels); err != nil {
		return nil
	}
	seen := make(map[string]struct{}, len(channels))
	result := make([]string, 0, len(channels))
	for _, channel := range channels {
		channel = strings.TrimSpace(channel)
		if channel == "" {
			continue
		}
		if _, ok := seen[channel]; ok {
			continue
		}
		seen[channel] = struct{}{}
		result = append(result, channel)
	}
	return result
}
