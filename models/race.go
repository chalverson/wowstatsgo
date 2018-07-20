package models

import (
	"database/sql"
)

// Mapping of the Race table.
type Race struct {
	Id   int64
	Mask int64
	Side string
	Name string
}

func (db *WowDB) GetRaceById(id int64) (*Race, error) {
	var dbRace Race
	err := db.QueryRow("SELECT id, mask, side, name FROM races WHERE id = $1", id).Scan(&dbRace.Id, &dbRace.Mask, &dbRace.Side, &dbRace.Name)
	switch {
	case err == sql.ErrNoRows:
		return &Race{}, err
	}
	return &dbRace, nil
}

func (db *WowDB) InsertRace(race *Race) error {
	_, err := db.Exec("INSERT INTO races (id, mask, side, name) VALUES ($1, $2, $3, $4)", race.Id, race.Mask, race.Side, race.Name)
	if err != nil {
		return err
	}
	return nil
}
