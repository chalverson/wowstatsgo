package main

import (
	"github.com/chalverson/wowstatsgo/models"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

// Defines database functions.
type Datastore interface {
	InsertToon(toon *models.Toon) error
	GetToonById(id int64) (*models.Toon, error)
	GetAllToons() []models.Toon
	InsertStats(stats *models.Stat) error
	GetAllToonLatestQuickSummary() ([]models.Stat, error)
	InsertRace(race *models.Race) error
	GetRaceById(id int64) (*models.Race, error)
	InsertToonClass(toonClass *models.ToonClass) error
	GetToonClassById(id int64) (*models.ToonClass, error)
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
func (db *WowDB) GetToonById(id int64) (*models.Toon, error) {
	var toon models.Toon
	dbRet := db.First(&toon, id)
	return &toon, dbRet.Error
}

// Get all Toons from the database
func (db *WowDB) GetAllToons() []models.Toon {
	var toons []models.Toon

	db.Find(&toons)

	return toons
}

// Insert a new Toon into the database. Does not need an ID as the database should handle entering it.
func (db *WowDB) InsertToon(toon *models.Toon) error {
	return db.Create(toon).Error
}

// Insert a stats record. This doesn't check to see if a duplicate exists, it relies on the database's
// constraints to handle that.
func (db *WowDB) InsertStats(stats *models.Stat) error {
	return db.Create(stats).Error
}

// Get a list of the latest Stat for all toons. This will get just the latest day's stats which is useful
// for email or CLI.
func (db *WowDB) GetAllToonLatestQuickSummary() ([]models.Stat, error) {
	var stats []models.Stat
	//db.LogMode(true)
	dbRet := db.Preload("Toon").Where("insert_date = (select max(insert_date) from stats)").Order("level desc").Order("item_level desc").Find(&stats)
	//db.LogMode(false)

	return stats, dbRet.Error
}

func (db *WowDB) InsertToonClass(toonClass *models.ToonClass) error {
	return db.Create(toonClass).Error
}

func (db *WowDB) GetToonClassById(id int64) (*models.ToonClass, error) {
	var dbClass models.ToonClass
	dbRet := db.First(&dbClass, id)
	return &dbClass, dbRet.Error
}

func (db *WowDB) GetRaceById(id int64) (*models.Race, error) {
	var dbRace models.Race
	dbRet := db.First(&dbRace, id)
	return &dbRace, dbRet.Error
}

func (db *WowDB) InsertRace(race *models.Race) error {
	return db.Create(race).Error
}
