package main

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
	GetAllToonLatestQuickSummary() ([]Stat, error)
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

// Get a Toon from the database based on Id
func (db *WowDB) GetToonById(id int64) (*Toon, error) {
	var toon Toon
	dbRet := db.First(&toon, id)
	return &toon, dbRet.Error
}

// Get all Toons from the database
func (db *WowDB) GetAllToons() []Toon {
	var toons []Toon

	db.Find(&toons)

	return toons
}

// Insert a new Toon into the database. Does not need an ID as the database should handle entering it.
func (db *WowDB) InsertToon(toon *Toon) error {
	return db.Create(toon).Error
}

// Insert a stats record. This doesn't check to see if a duplicate exists, it relies on the database's
// constraints to handle that.
func (db *WowDB) InsertStats(stats *Stat) error {
	return db.Create(stats).Error
}

// Get a list of the latest Stat for all toons. This will get just the latest day's stats which is useful
// for email or CLI.
func (db *WowDB) GetAllToonLatestQuickSummary() ([]Stat, error) {
	var stats []Stat
	//db.LogMode(true)
	dbRet := db.Preload("Toon").Where("insert_date = (select max(insert_date) from stats)").Order("level desc").Order("item_level desc").Find(&stats)
	//db.LogMode(false)

	return stats, dbRet.Error
}

func (db *WowDB) InsertToonClass(toonClass *ToonClass) error {
	return db.Create(toonClass).Error
}

func (db *WowDB) GetToonClassById(id int64) (*ToonClass, error) {
	var dbClass ToonClass
	dbRet := db.First(&dbClass, id)
	return &dbClass, dbRet.Error
}

func (db *WowDB) GetRaceById(id int64) (*Race, error) {
	var dbRace Race
	dbRet := db.First(&dbRace, id)
	return &dbRace, dbRet.Error
}

func (db *WowDB) InsertRace(race *Race) error {
	return db.Create(race).Error
}
