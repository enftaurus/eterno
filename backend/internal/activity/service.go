package activity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/eterno/backend/internal/models"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) RecordEvent(ctx context.Context, userID, source, externalID, eventType string, occurredAt time.Time, metadata map[string]interface{}) error {
	metaJSON, _ := json.Marshal(metadata)
	query := `
		INSERT INTO activity_events (user_id, source, external_id, event_type, occurred_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (source, external_id) DO NOTHING
	`
	_, err := s.db.ExecContext(ctx, query, userID, source, externalID, eventType, occurredAt, metaJSON)
	return err
}

func (s *Service) GetUserHeatmap(ctx context.Context, userID string, year int, sourceFilter string) (*models.HeatmapResponse, error) {
	if year <= 0 {
		year = time.Now().UTC().Year()
	}

	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)

	query := `
		SELECT 
			DATE(occurred_at)::text AS day_date,
			source,
			COUNT(id)::int AS event_count
		FROM activity_events
		WHERE user_id = $1
		  AND occurred_at >= $2
		  AND occurred_at <= $3
	`
	args := []interface{}{userID, startDate, endDate}
	if sourceFilter != "" {
		query += " AND source = $4"
		args = append(args, sourceFilter)
	}
	query += `
		GROUP BY DATE(occurred_at), source
		ORDER BY day_date ASC
	`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query activity heatmap: %w", err)
	}
	defer rows.Close()

	dayMap := make(map[string]*models.HeatmapDayData)
	totalEvents := 0

	for rows.Next() {
		var dayDate, source string
		var count int
		if err := rows.Scan(&dayDate, &source, &count); err != nil {
			return nil, err
		}

		if _, exists := dayMap[dayDate]; !exists {
			dayMap[dayDate] = &models.HeatmapDayData{
				Date:            dayDate,
				Count:           0,
				Intensity:       0,
				SourceBreakdown: make(map[string]int),
			}
		}

		dayMap[dayDate].Count += count
		dayMap[dayDate].SourceBreakdown[source] += count
		totalEvents += count
	}

	// Calculate intensities
	days := make([]models.HeatmapDayData, 0, len(dayMap))
	for _, d := range dayMap {
		d.Intensity = CalculateIntensity(d.Count)
		days = append(days, *d)
	}

	return &models.HeatmapResponse{
		TotalEvents: totalEvents,
		Days:        days,
	}, nil
}

func CalculateIntensity(count int) int {
	switch {
	case count <= 0:
		return 0
	case count <= 2:
		return 1
	case count <= 5:
		return 2
	case count <= 9:
		return 3
	default:
		return 4
	}
}
