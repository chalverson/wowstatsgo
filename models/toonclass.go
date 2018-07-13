package models

import (
	"database/sql"
)

// Map the classes table, but call it ToonClass here.
type ToonClass struct {
	Id        int64
	Mask      int64
	PowerType string
	Name      string
}

func (db *WowDB) InsertToonClass(toonClass *ToonClass) error {
	_, err := db.Exec("INSERT INTO classes (id, mask, powerType, name) VALUES ($1, $2, $3, $4)", toonClass.Id, toonClass.Mask, toonClass.PowerType, toonClass.Name)
	if err != nil {
		return err
	}
	return nil
}

func (db *WowDB) GetToonClassById(id int64) (*ToonClass, error) {
	var dbClass ToonClass
	err := db.QueryRow("SELECT id, mask, powerType, name FROM classes WHERE id = $1", id).Scan(&dbClass.Id, &dbClass.Mask, &dbClass.PowerType, &dbClass.Name)
	switch {
	case err == sql.ErrNoRows:
		return &ToonClass{}, err
	}
	return &dbClass, nil
}
