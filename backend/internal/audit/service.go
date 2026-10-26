package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/eterno/backend/internal/models"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Log(ctx context.Context, institutionID, actorID *string, action, entityType, entityID string, details map[string]interface{}, ipAddress *string) error {
	detailsJSON, _ := json.Marshal(details)
	query := `
		INSERT INTO audit_logs (institution_id, actor_id, action, entity_type, entity_id, details, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`
	var instID, actID, ip sql.NullString
	if institutionID != nil && *institutionID != "" {
		instID.String = *institutionID
		instID.Valid = true
	}
	if actorID != nil && *actorID != "" {
		actID.String = *actorID
		actID.Valid = true
	}
	if ipAddress != nil && *ipAddress != "" {
		ip.String = *ipAddress
		ip.Valid = true
	}

	_, err := s.db.ExecContext(ctx, query, instID, actID, action, entityType, entityID, detailsJSON, ip)
	return err
}

func (s *Service) List(ctx context.Context, institutionID string, limit int) ([]models.AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, institution_id, actor_id, action, entity_type, entity_id, details, ip_address, created_at
		FROM audit_logs
		WHERE institution_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := s.db.QueryContext(ctx, query, institutionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var log models.AuditLog
		var instID, actID, ip sql.NullString
		var detailsBytes []byte
		if err := rows.Scan(
			&log.ID, &instID, &actID, &log.Action, &log.EntityType, &log.EntityID,
			&detailsBytes, &ip, &log.CreatedAt,
		); err != nil {
			return nil, err
		}
		if instID.Valid {
			log.InstitutionID = &instID.String
		}
		if actID.Valid {
			log.ActorID = &actID.String
		}
		if ip.Valid {
			log.IPAddress = &ip.String
		}
		log.Details = detailsBytes
		logs = append(logs, log)
	}
	return logs, nil
}
