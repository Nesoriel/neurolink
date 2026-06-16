package statusapi

import (
	"context"
	"time"
)

type DemoProvider struct{}

func NewDemoProvider() DemoProvider {
	return DemoProvider{}
}

func (DemoProvider) Fetch(ctx context.Context) (Snapshot, error) {
	select {
	case <-ctx.Done():
		return Snapshot{}, ctx.Err()
	default:
	}

	now := time.Now()
	snapshot := Snapshot{
		Source:      SourceDemo,
		Attribution: Attribution,
		GeneratedAt: now,
		Overall:     StatusDegraded,
		Notice:      "Demo mode: deterministic sample data, not live Apex service status",
		Services: []ServiceStatus{
			demoService(ServiceCrossplayAuth, "Crossplay Auth", StatusHealthy, "6 demo regions running", now, demoRegions(StatusHealthy, 35*time.Millisecond, now)),
			demoService(ServiceMatchmaking, "Lobby / Matchmaking", StatusDegraded, "1 demo region degraded, 6 running", now, demoMatchmakingRegions(now)),
			demoService(ServicePCLogin, "PC / Desktop Logins", StatusHealthy, "6 demo regions running", now, demoRegions(StatusHealthy, 18*time.Millisecond, now)),
			demoService(ServicePlayerAccount, "Player Accounts", StatusHealthy, "6 demo regions running", now, demoRegions(StatusHealthy, 41*time.Millisecond, now)),
			demoService(ServiceAPIHealth, "Apex Legends Status API", StatusHealthy, "Demo API health is simulated", now, []RegionStatus{{Name: "Status API", Status: StatusHealthy, Label: "RUNNING", Latency: 64 * time.Millisecond, HasLatency: true, CheckedAt: now}}),
		},
		RecentReports: []RecentReport{
			{Country: "US", At: "demo", Issue: "Connectivity", Platform: "Steam", ErrorCode: "None"},
			{Country: "JP", At: "demo", Issue: "Match making", Platform: "PC", ErrorCode: "Net"},
		},
	}
	return snapshot, nil
}

func demoService(id ServiceID, name string, status Status, summary string, updatedAt time.Time, regions []RegionStatus) ServiceStatus {
	return ServiceStatus{
		ID:        id,
		Name:      name,
		Status:    status,
		Summary:   summary,
		UpdatedAt: updatedAt,
		Regions:   regions,
	}
}

func demoMatchmakingRegions(now time.Time) []RegionStatus {
	regions := demoRegions(StatusHealthy, 22*time.Millisecond, now)
	return append(regions, RegionStatus{
		Name:       "Asia",
		Status:     StatusDegraded,
		Label:      "SLOW",
		Latency:    140 * time.Millisecond,
		HasLatency: true,
		CheckedAt:  now,
	})
}

func demoRegions(status Status, latency time.Duration, now time.Time) []RegionStatus {
	names := []string{"EU West", "EU East", "US West", "US Central", "US East", "South America"}
	regions := make([]RegionStatus, 0, len(names))
	for i, name := range names {
		regions = append(regions, RegionStatus{
			Name:       name,
			Status:     status,
			Label:      "RUNNING",
			Latency:    latency + time.Duration(i*3)*time.Millisecond,
			HasLatency: true,
			CheckedAt:  now,
		})
	}
	return regions
}
