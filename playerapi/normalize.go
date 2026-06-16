package playerapi

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

func NormalizeBridgeResponse(raw map[string]any, request LookupRequest, source SourceMode, now time.Time) PlayerSnapshot {
	if now.IsZero() {
		now = time.Now()
	}
	request, _ = request.normalized()

	data := objectAt(raw, "data")
	if len(data) == 0 {
		data = raw
	}
	global := objectAt(data, "global")
	if len(global) == 0 {
		global = data
	}

	snapshot := PlayerSnapshot{
		Source:         source,
		Attribution:    Attribution,
		LookupAt:       now,
		PlayerName:     firstNonEmpty(firstString(global, "name", "playerName", "username", "displayName"), request.Player),
		Platform:       normalizeResponsePlatform(firstString(global, "platform", "platformSlug"), request.Platform),
		UID:            firstString(global, "uid", "UID", "id", "userId"),
		SelectedLegend: normalizeSelectedLegend(data),
		Trackers:       normalizeTrackers(data),
	}

	if level, ok := firstInt(global, "level", "Level"); ok {
		snapshot.Level = level
		snapshot.HasLevel = true
	}
	if rank, ok := normalizeRank(global); ok {
		snapshot.RankName = rank.name
		snapshot.RankDivision = rank.division
		snapshot.RankScore = rank.score
		snapshot.HasRank = true
	}
	return snapshot
}

type normalizedRank struct {
	name     string
	division string
	score    int
}

func normalizeResponsePlatform(value string, fallback Platform) Platform {
	platform, err := ParsePlatform(value)
	if err == nil {
		return platform
	}
	return fallback
}

func normalizeRank(global map[string]any) (normalizedRank, bool) {
	rankObject := objectAt(global, "rank")
	if len(rankObject) == 0 {
		rankObject = objectAt(global, "Rank")
	}
	if len(rankObject) == 0 {
		return normalizedRank{}, false
	}

	rank := normalizedRank{
		name:     firstString(rankObject, "rankName", "RankName", "name", "Name"),
		division: firstString(rankObject, "rankDiv", "RankDiv", "division", "Division"),
	}
	if score, ok := firstInt(rankObject, "rankScore", "RankScore", "score", "Score"); ok {
		rank.score = score
	}
	return rank, rank.name != "" || rank.division != "" || rank.score > 0
}

func normalizeSelectedLegend(data map[string]any) string {
	legends := objectAt(data, "legends")
	selected := objectAt(legends, "selected")
	if len(selected) == 0 {
		selected = objectAt(data, "selected")
	}
	return firstString(selected, "LegendName", "legendName", "name", "Name")
}

func normalizeTrackers(data map[string]any) []Tracker {
	trackersByName := map[string]Tracker{}
	for _, tracker := range trackersFromObject(objectAt(data, "total")) {
		trackersByName[normalizeTrackerKey(tracker.Name)] = tracker
	}

	legends := objectAt(data, "legends")
	selected := objectAt(legends, "selected")
	for _, tracker := range trackersFromArray(selected["data"]) {
		key := normalizeTrackerKey(tracker.Name)
		if _, exists := trackersByName[key]; !exists {
			trackersByName[key] = tracker
		}
	}

	if len(trackersByName) == 0 {
		return nil
	}
	trackers := make([]Tracker, 0, len(trackersByName))
	for _, tracker := range trackersByName {
		trackers = append(trackers, tracker)
	}
	sort.Slice(trackers, func(i, j int) bool {
		return strings.ToLower(trackers[i].Name) < strings.ToLower(trackers[j].Name)
	})
	return trackers
}

func trackersFromObject(object map[string]any) []Tracker {
	if len(object) == 0 {
		return nil
	}
	trackers := make([]Tracker, 0, len(object))
	for key, raw := range object {
		tracker, ok := normalizeTracker(key, raw)
		if ok {
			trackers = append(trackers, tracker)
		}
	}
	return trackers
}

func trackersFromArray(raw any) []Tracker {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	trackers := make([]Tracker, 0, len(items))
	for _, item := range items {
		tracker, ok := normalizeTracker("", item)
		if ok {
			trackers = append(trackers, tracker)
		}
	}
	return trackers
}

func normalizeTracker(key string, raw any) (Tracker, bool) {
	object, ok := raw.(map[string]any)
	if !ok {
		value := formatValue(raw)
		name := humanizeKey(key)
		return Tracker{Name: name, Value: value}, name != "" && value != ""
	}

	name := firstString(object, "name", "Name", "key", "Key", "metadataName")
	if name == "" {
		name = humanizeKey(key)
	}
	value := firstValue(object, "value", "Value", "total", "Total")
	displayValue := formatValue(value)
	return Tracker{Name: name, Value: displayValue}, name != "" && displayValue != ""
}

func normalizeTrackerKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func objectAt(data map[string]any, key string) map[string]any {
	value, ok := data[key]
	if !ok {
		return nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return object
}

func firstString(data map[string]any, keys ...string) string {
	value := firstValue(data, keys...)
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case bool:
		return strconv.FormatBool(v)
	default:
		return ""
	}
}

func firstInt(data map[string]any, keys ...string) (int, bool) {
	value := firstValue(data, keys...)
	switch v := value.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case jsonNumber:
		i, err := strconv.Atoi(string(v))
		return i, err == nil
	case string:
		clean := strings.ReplaceAll(strings.TrimSpace(v), ",", "")
		i, err := strconv.Atoi(clean)
		if err == nil {
			return i, true
		}
		f, err := strconv.ParseFloat(clean, 64)
		if err == nil {
			return int(f), true
		}
	}
	return 0, false
}

type jsonNumber string

func firstValue(data map[string]any, keys ...string) any {
	for _, key := range keys {
		value, ok := data[key]
		if ok {
			return value
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func formatValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', 1, 64)
	case int:
		return strconv.Itoa(v)
	case bool:
		return strconv.FormatBool(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func humanizeKey(value string) string {
	value = strings.ReplaceAll(value, "_", " ")
	value = strings.ReplaceAll(value, "-", " ")
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Fields(value)
	for i, part := range parts {
		if len(part) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
