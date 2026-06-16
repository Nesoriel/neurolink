package statusapi

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type serviceAlias struct {
	id      ServiceID
	name    string
	aliases []string
}

var serviceAliases = []serviceAlias{
	{id: ServiceCrossplayAuth, name: "Crossplay Auth", aliases: []string{"ApexOauth_Crossplay", "ApexOauth Crossplay", "Crossplay auth", "Crossplay"}},
	{id: ServiceMatchmaking, name: "Lobby / Matchmaking", aliases: []string{"EA_novafusin", "EA_novafusion", "Lobby/Matchmaking servers", "Lobby Matchmaking", "Matchmaking"}},
	{id: ServicePCLogin, name: "PC / Desktop Logins", aliases: []string{"Origin_login", "PC/Desktop logins", "PC Desktop logins", "Origin login"}},
	{id: ServicePlayerAccount, name: "Player Accounts", aliases: []string{"EA_accounts", "Player accounts", "Accounts"}},
	{id: ServiceAPIHealth, name: "Apex Legends Status API", aliases: []string{"selfCoreTest", "ALS website", "Status website", "API health"}},
}

func NormalizeServerStatus(raw map[string]any, source SourceMode, now time.Time) Snapshot {
	if now.IsZero() {
		now = time.Now()
	}

	data := objectAt(raw, "data")
	if len(data) == 0 {
		data = raw
	}

	services := make([]ServiceStatus, 0, len(serviceAliases))
	for _, alias := range serviceAliases {
		service, ok := normalizeService(data, alias)
		if !ok {
			service = ServiceStatus{
				ID:        alias.id,
				Name:      alias.name,
				Status:    StatusUnknown,
				Summary:   "No status data in API response",
				UpdatedAt: now,
			}
			if alias.id == ServiceAPIHealth && source == SourceLive {
				service.Status = StatusHealthy
				service.Summary = "Server status endpoint responded; self-test payload not provided"
			}
		}
		services = append(services, service)
	}

	snapshot := Snapshot{
		Source:        source,
		Attribution:   Attribution,
		GeneratedAt:   now,
		Services:      services,
		RecentReports: normalizeReports(data),
	}
	snapshot.Overall = overallStatus(services)
	return snapshot
}

func UnavailableSnapshot(source SourceMode, err error, now time.Time) Snapshot {
	if now.IsZero() {
		now = time.Now()
	}

	services := CoreServices()
	for i := range services {
		services[i].UpdatedAt = now
		if services[i].ID == ServiceAPIHealth {
			services[i].Status = StatusDown
			services[i].Summary = "Status API request failed"
		}
	}

	message := ""
	if err != nil {
		message = err.Error()
	}

	return Snapshot{
		Source:      source,
		Attribution: Attribution,
		GeneratedAt: now,
		Overall:     StatusUnknown,
		Services:    services,
		Notice:      "Unable to refresh service data",
		LastError:   message,
	}
}

func NormalizeStatus(value string) Status {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "_", " ")
	normalized = strings.ReplaceAll(normalized, "-", " ")
	if normalized == "" {
		return StatusUnknown
	}

	switch {
	case strings.Contains(normalized, "offline"),
		strings.Contains(normalized, "outage"),
		strings.Contains(normalized, "unavailable"),
		strings.Contains(normalized, "down"),
		strings.Contains(normalized, "fail"),
		normalized == "ko":
		return StatusDown
	case strings.Contains(normalized, "degraded"),
		strings.Contains(normalized, "partial"),
		strings.Contains(normalized, "slow"),
		strings.Contains(normalized, "unstable"),
		strings.Contains(normalized, "maintenance"),
		strings.Contains(normalized, "minor"):
		return StatusDegraded
	case strings.Contains(normalized, "running"),
		strings.Contains(normalized, "operational"),
		strings.Contains(normalized, "online"),
		normalized == "up",
		normalized == "ok",
		normalized == "healthy":
		return StatusHealthy
	default:
		return StatusUnknown
	}
}

func normalizeService(data map[string]any, alias serviceAlias) (ServiceStatus, bool) {
	key, value, ok := findAliasedValue(data, alias.aliases)
	if !ok {
		return ServiceStatus{}, false
	}

	regions := normalizeRegions(value)
	service := ServiceStatus{
		ID:      alias.id,
		Name:    alias.name,
		Status:  aggregateRegions(regions),
		Regions: regions,
	}

	service.UpdatedAt = latestRegionTime(regions)
	if service.UpdatedAt.IsZero() {
		service.UpdatedAt = time.Now()
	}
	service.Summary = summarizeService(service, key)
	return service, true
}

func normalizeRegions(value any) []RegionStatus {
	object, ok := value.(map[string]any)
	if !ok {
		return nil
	}

	regions := make([]RegionStatus, 0, len(object))
	for regionName, rawRegion := range object {
		region, ok := normalizeRegion(regionName, rawRegion)
		if ok {
			regions = append(regions, region)
		}
	}

	sort.Slice(regions, func(i, j int) bool {
		return regions[i].Name < regions[j].Name
	})
	return regions
}

func normalizeRegion(name string, raw any) (RegionStatus, bool) {
	region := RegionStatus{Name: humanizeKey(name), Status: StatusUnknown}
	object, ok := raw.(map[string]any)
	if !ok {
		region.Label = strings.TrimSpace(fmt.Sprint(raw))
		region.Status = NormalizeStatus(region.Label)
		return region, region.Label != ""
	}

	label := firstString(object, "Status", "status", "state", "State", "message", "Message")
	status := NormalizeStatus(label)
	if status == StatusUnknown {
		status = statusFromHTTPCode(firstFloat(object, "HTTPCode", "httpCode", "code", "Code"))
	}

	region.Status = status
	region.Label = label

	if latency, ok := firstFloat(object, "ResponseTime", "responseTime", "latency", "Latency", "ping", "Ping"); ok {
		region.Latency = time.Duration(latency * float64(time.Millisecond))
		region.HasLatency = true
	}
	if uptime, ok := firstFloat(object, "Uptime", "uptime", "uptimePercent", "UptimePercent"); ok {
		region.UptimePercent = uptime
		region.HasUptime = true
	}
	if checkedAt := firstFloatValue(object, "QueryTimestamp", "queryTimestamp", "timestamp", "Timestamp"); checkedAt > 0 {
		region.CheckedAt = time.Unix(int64(checkedAt), 0)
	}

	return region, true
}

func statusFromHTTPCode(code float64, ok bool) Status {
	if !ok {
		return StatusUnknown
	}
	switch {
	case code >= 200 && code < 300:
		return StatusHealthy
	case code == 429 || (code >= 400 && code < 500):
		return StatusDegraded
	case code >= 500:
		return StatusDown
	default:
		return StatusUnknown
	}
}

func aggregateRegions(regions []RegionStatus) Status {
	if len(regions) == 0 {
		return StatusUnknown
	}

	overall := StatusHealthy
	for _, region := range regions {
		switch region.Status {
		case StatusDown:
			return StatusDown
		case StatusDegraded:
			overall = StatusDegraded
		case StatusUnknown:
			if overall == StatusHealthy {
				overall = StatusUnknown
			}
		}
	}
	return overall
}

func overallStatus(services []ServiceStatus) Status {
	if len(services) == 0 {
		return StatusUnknown
	}

	overall := StatusHealthy
	for _, service := range services {
		if service.Status == StatusDown {
			return StatusDown
		}
		if service.Status == StatusDegraded {
			overall = StatusDegraded
			continue
		}
		if service.Status == StatusUnknown && overall == StatusHealthy {
			overall = StatusUnknown
		}
	}
	return overall
}

func summarizeService(service ServiceStatus, sourceKey string) string {
	if len(service.Regions) == 0 {
		return "No regional checks reported"
	}

	counts := map[Status]int{}
	for _, region := range service.Regions {
		counts[region.Status]++
	}

	switch service.Status {
	case StatusHealthy:
		return fmt.Sprintf("%d checks running", len(service.Regions))
	case StatusDegraded:
		return fmt.Sprintf("%d degraded, %d running", counts[StatusDegraded]+counts[StatusUnknown], counts[StatusHealthy])
	case StatusDown:
		return fmt.Sprintf("%d down, %d running", counts[StatusDown], counts[StatusHealthy])
	default:
		return fmt.Sprintf("%s reported %d checks with unknown state", humanizeKey(sourceKey), len(service.Regions))
	}
}

func latestRegionTime(regions []RegionStatus) time.Time {
	var latest time.Time
	for _, region := range regions {
		if region.CheckedAt.After(latest) {
			latest = region.CheckedAt
		}
	}
	return latest
}

func normalizeReports(data map[string]any) []RecentReport {
	for _, key := range []string{"lastReports", "reports", "recentReports", "LastReports"} {
		raw, ok := data[key]
		if !ok {
			continue
		}
		items, ok := raw.([]any)
		if !ok {
			continue
		}
		reports := make([]RecentReport, 0, len(items))
		for _, item := range items {
			object, ok := item.(map[string]any)
			if !ok {
				continue
			}
			reports = append(reports, RecentReport{
				Country:   firstString(object, "country", "Country"),
				At:        firstString(object, "date", "Date", "time", "Time"),
				Issue:     firstString(object, "issue", "Issue", "type", "Type"),
				Platform:  firstString(object, "platform", "Platform"),
				ErrorCode: firstString(object, "errorCode", "ErrorCode", "code", "Code"),
			})
		}
		return reports
	}
	return nil
}

func findAliasedValue(data map[string]any, aliases []string) (string, any, bool) {
	for _, alias := range aliases {
		if value, ok := data[alias]; ok {
			return alias, value, true
		}
	}

	normalizedAliases := make(map[string]struct{}, len(aliases))
	for _, alias := range aliases {
		normalizedAliases[normalizeKey(alias)] = struct{}{}
	}
	for key, value := range data {
		if _, ok := normalizedAliases[normalizeKey(key)]; ok {
			return key, value, true
		}
	}
	return "", nil, false
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
	for _, key := range keys {
		value, ok := data[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case string:
			return strings.TrimSpace(v)
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64)
		case bool:
			return strconv.FormatBool(v)
		}
	}
	return ""
}

func firstFloat(data map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		value, ok := data[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case float64:
			return v, true
		case int:
			return float64(v), true
		case string:
			parsed, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(v), "%"), 64)
			if err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func firstFloatValue(data map[string]any, keys ...string) float64 {
	value, _ := firstFloat(data, keys...)
	return value
}

func normalizeKey(value string) string {
	value = strings.ToLower(value)
	replacer := strings.NewReplacer("_", "", "-", "", "/", "", " ", "", "#", "")
	return replacer.Replace(value)
}

func humanizeKey(value string) string {
	value = strings.ReplaceAll(value, "_", " ")
	value = strings.ReplaceAll(value, "-", " ")
	value = strings.ReplaceAll(value, "#", "#")
	value = strings.TrimSpace(value)
	if value == "" {
		return "Unknown"
	}

	parts := strings.Fields(value)
	for i, part := range parts {
		switch strings.ToUpper(part) {
		case "API", "PC", "EU", "US", "ALS":
			parts[i] = strings.ToUpper(part)
		default:
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}
