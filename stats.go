package main

import (
	"github.com/jinzhu/gorm"
	"time"
)

// Map the stats table. This holds the "interesting" stats that I'm interested in.
type Stat struct {
	gorm.Model
	Toon              Toon
	ToonID            uint
	LastModified      int64
	InsertDate        time.Time `gorm:"type:date"`
	Level             int64
	AchievementPoints int64
	ExaltedReps       int64
	MountsCollected   int64
	QuestsCompleted   int64
	FishCaught        int64
	PetsCollected     int64
	PetBattlesWon     int64
	PetBattlesPvpWon  int64
	ItemLevel         int64
	HonorableKills    int64
}

// Get the LastModified field as a human readable format as YYYY-MM-DD HH:MM:SS.
// Need to pull this out separately because we have extra detail from the JSON. We need to divide the time
// by 1000, then format it.
func (s *Stat) LastModifiedAsDateTime() string {
	t := time.Unix(s.LastModified/1000, 0)
	return t.UTC().Format("2006-01-02 15:04:05")
}
