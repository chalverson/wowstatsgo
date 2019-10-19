package main

import (
	"github.com/jinzhu/gorm"
)

// Database table model
type Toon struct {
	gorm.Model
	Name      string
	Race      Race `gorm:"foreignkey:RaceID"`
	RaceID    int64
	ToonClass ToonClass `gorm:"foreignkey:ClassID"`
	ClassID   int64
	Gender    int64
	Realm     string
	Region    string
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
