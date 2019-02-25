package models

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

// Defines database functions.
type Datastore interface {
	InsertToon(toon *Toon) error
	GetToonById(id int64) (*Toon, error)
	GetAllToons() []Toon
	InsertStats(stats *Stat) error
	GetAllToonLatestQuickSummary() []Stat
	InsertRace(race *Race) error
	GetRaceById(id int64) (*Race, error)
	InsertToonClass(toonClass *ToonClass) error
	GetToonClassById(id int64) (*ToonClass, error)
}

// Database interface struct.
type WowDB struct {
	*gorm.DB
	dbDriver string
}

func NewDB(dbDriver string, connStr string) (*WowDB, error) {
	db, err := gorm.Open(dbDriver, connStr)
	if err != nil {
		return nil, err
	}

	err = db.DB().Ping()
	if err != nil {
		return nil, err
	}

	db.Set("gorm:auto_preload", true)
	//db.LogMode(true)
	return &WowDB{db, dbDriver}, nil
}
