package util

import "database/sql"

type Initiator struct {
	UtilRepository
	ApiFormaters
}

func NewInitiator(sqlDB *sql.DB) *Initiator {
	formaterRepo := UtilRepository{
		sqlDB,
	}
	return &Initiator{
		UtilRepository{
			sql: sqlDB,
		},
		ApiFormaters{
			Repo: formaterRepo,
		},
	}
}
