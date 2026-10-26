package groups

import (
	"context"
	"database/sql"
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

func (s *Service) CreateGroup(ctx context.Context, institutionID, name, description, groupType, createdBy string) (*models.Group, error) {
	query := `
		INSERT INTO groups (institution_id, name, description, group_type, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, institution_id, name, description, group_type, created_by, created_at
	`
	var g models.Group
	var desc, creator sql.NullString
	if description != "" {
		desc.String = description
		desc.Valid = true
	}
	if createdBy != "" {
		creator.String = createdBy
		creator.Valid = true
	}

	err := s.db.QueryRowContext(ctx, query, institutionID, name, desc, groupType, creator).Scan(
		&g.ID, &g.InstitutionID, &g.Name, &desc, &g.GroupType, &creator, &g.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer group: %w", err)
	}
	if desc.Valid {
		g.Description = &desc.String
	}
	if creator.Valid {
		g.CreatedBy = &creator.String
	}
	return &g, nil
}

func (s *Service) ListGroupsForInstitution(ctx context.Context, institutionID string) ([]map[string]interface{}, error) {
	query := `
		SELECT g.id, g.institution_id, g.name, g.description, g.group_type, g.created_at,
		       COUNT(gm.user_id)::int AS member_count,
		       COALESCE(SUM(se.points), 0)::int AS total_group_points
		FROM groups g
		LEFT JOIN group_memberships gm ON g.id = gm.group_id
		LEFT JOIN score_events se ON gm.user_id = se.user_id AND se.institution_id = g.institution_id
		WHERE g.institution_id = $1
		GROUP BY g.id, g.institution_id, g.name, g.description, g.group_type, g.created_at
		ORDER BY total_group_points DESC, g.name ASC
	`
	rows, err := s.db.QueryContext(ctx, query, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var id, instID, name, gType string
		var desc sql.NullString
		var createdAt time.Time
		var memberCount, totalPoints int

		if err := rows.Scan(&id, &instID, &name, &desc, &gType, &createdAt, &memberCount, &totalPoints); err != nil {
			return nil, err
		}

		item := map[string]interface{}{
			"id":                 id,
			"institution_id":     instID,
			"name":               name,
			"group_type":         gType,
			"created_at":         createdAt,
			"member_count":       memberCount,
			"total_group_points": totalPoints,
		}
		if desc.Valid {
			item["description"] = desc.String
		}
		if memberCount > 0 {
			item["points_per_student"] = float64(totalPoints) / float64(memberCount)
		} else {
			item["points_per_student"] = 0.0
		}
		list = append(list, item)
	}
	return list, nil
}

func (s *Service) AddMember(ctx context.Context, groupID, userID, role string) error {
	if role == "" {
		role = "MEMBER"
	}
	query := `
		INSERT INTO group_memberships (group_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (group_id, user_id) DO NOTHING
	`
	_, err := s.db.ExecContext(ctx, query, groupID, userID, role)
	return err
}
