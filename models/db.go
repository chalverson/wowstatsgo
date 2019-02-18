package models

import (
	_ "github.com/lib/pq"
	_ "github.com/go-sql-driver/mysql"
	"database/sql"
)

type Datastore interface {
	InsertToon(toon *Toon) error
	GetToonById(id int64) (*Toon, error)
	GetAllToons() []Toon
	InsertStats(stats *Stats) error
	GetAllToonLatestQuickSummary() []Stats
	InsertRace(race *Race) error
	GetRaceById(id int64) (*Race, error)
	InsertToonClass(toonClass *ToonClass) error
	GetToonClassById(id int64) (*ToonClass, error)
}

type WowDB struct {
	*sql.DB
}

func NewDB(dbDriver string, connStr string) (*WowDB, error) {
	db, err := sql.Open(dbDriver, connStr)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}
	return &WowDB{db}, nil
}
