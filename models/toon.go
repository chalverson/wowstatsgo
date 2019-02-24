package models

import "github.com/jinzhu/gorm"

// Database table model
type Toon struct {
	gorm.Model
	Name    string
	Race    Race `gorm:"foreignkey:RaceID"`
	RaceID  int64
	ToonClass ToonClass `gorm:"foreignkey:ClassID"`
	ClassID int64
	Gender  int64
	Realm   string
	Region  string
}

type ToonDto struct {
	Name    string
	RaceID  int64
	ClassID int64
	Gender  int64
	Realm   string
	Region  string
}


// Create a new ToonDto struct
func NewToon(name string, race int64, class int64, gender int64, realm string, region string) *ToonDto {
	return &ToonDto{
		Name:    name,
		RaceID:  race,
		ClassID: class,
		Gender:  gender,
		Realm:   realm,
		Region:  region,
	}
}

// Get a Toon from the database based on Id
func (db *WowDB) GetToonById(id int64) (*Toon, error) {
	var toon Toon
	db.First(&toon, id)
	return &toon, nil
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
