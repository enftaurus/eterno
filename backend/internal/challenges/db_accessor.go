package challenges

import (
	"database/sql"
)

func (s *Service) DB() *sql.DB {
	return s.db
}
