package playerapi

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type DemoProvider struct{}

func NewDemoProvider() DemoProvider {
	return DemoProvider{}
}

func (DemoProvider) Lookup(ctx context.Context, request LookupRequest) (PlayerSnapshot, error) {
	select {
	case <-ctx.Done():
		return PlayerSnapshot{}, ctx.Err()
	default:
	}

	request, err := request.normalized()
	if err != nil {
		return PlayerSnapshot{}, err
	}

	name := request.Player
	key := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	return PlayerSnapshot{
		Source:         SourceDemo,
		Attribution:    Attribution,
		LookupAt:       time.Now(),
		PlayerName:     name,
		Platform:       request.Platform,
		UID:            "demo-" + key,
		Level:          412,
		HasLevel:       true,
		RankName:       "Gold",
		RankDivision:   "2",
		RankScore:      7350,
		HasRank:        true,
		SelectedLegend: "Crypto",
		Trackers: []Tracker{
			{Name: "Kills", Value: "1842"},
			{Name: "Damage", Value: "542931"},
			{Name: "Wins", Value: "104"},
		},
		Notice: fmt.Sprintf("Demo player lookup for %s on %s; not live Apex player data", name, request.Platform),
	}, nil
}
