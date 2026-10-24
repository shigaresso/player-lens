package adapter

import (
	"fmt"
	"strconv"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func parseRecommendationID(value string) (steamdomain.RecommendationID, error) {
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse Steam recommendationid: %w", err)
	}
	return steamdomain.RecommendationID(id), nil
}
