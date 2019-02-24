package models

// Mapping of the RaceID table.
type Race struct {
	ID   int64
	Mask int64
	Side string
	Name string
}

func (db *WowDB) GetRaceById(id int64) (*Race, error) {
	var dbRace Race
	dbRet := db.First(&dbRace, id)
	return &dbRace, dbRet.Error

	//var sqlString string
	//if db.dbDriver == "postgres" {
	//	sqlString = "SELECT id, mask, side, name FROM races WHERE id = $1"
	//} else if db.dbDriver == "mysql" {
	//	sqlString = "SELECT id, mask, side, name FROM races WHERE id = ?"
	//}
	//
	//err := db.QueryRow(sqlString, id).Scan(&dbRace.Id, &dbRace.Mask, &dbRace.Side, &dbRace.Name)
	//switch {
	//case err == sql.ErrNoRows:
	//	return &RaceID{}, err
	//}
	//return &dbRace, nil
}

func (db *WowDB) InsertRace(race *Race) error {
	return db.Create(race).Error

	//var sqlString string
	//if db.dbDriver == "postgres" {
	//	sqlString = "INSERT INTO races (id, mask, side, name) VALUES ($1, $2, $3, $4)"
	//} else if db.dbDriver == "mysql" {
	//	sqlString = "INSERT INTO races (id, mask, side, name) VALUES (?, ?, ?, ?)"
	//}
	//_, err := db.Exec(sqlString, race.Id, race.Mask, race.Side, race.Name)
	//if err != nil {
	//	return err
	//}
	//return nil
}
