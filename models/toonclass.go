package models

// Map the classes table, but call it ToonClass here.
type ToonClass struct {
	ID        int64
	Mask      int64
	PowerType string
	Name      string
}

func (db *WowDB) InsertToonClass(toonClass *ToonClass) error {
	return db.Create(toonClass).Error

	//var sqlString string
	//if db.dbDriver == "postgres" {
	//	sqlString = "INSERT INTO classes (id, mask, powerType, name) VALUES ($1, $2, $3, $4)"
	//} else if db.dbDriver == "mysql" {
	//	sqlString = "INSERT INTO classes (id, mask, powerType, name) VALUES (?, ?, ?, ?)"
	//}
	//
	//_, err := db.Exec(sqlString, toonClass.Id, toonClass.Mask, toonClass.PowerType, toonClass.Name)
	//if err != nil {
	//	return err
	//}
	//return nil
}

func (db *WowDB) GetToonClassById(id int64) (*ToonClass, error) {
	var dbClass ToonClass
	dbRet := db.First(&dbClass, id)
	return &dbClass, dbRet.Error

	//var sqlString string
	//if db.dbDriver == "postgres" {
	//	sqlString = "SELECT id, mask, powerType, name FROM classes WHERE id = $1"
	//} else if db.dbDriver == "mysql" {
	//	sqlString = "SELECT id, mask, powerType, name FROM classes WHERE id = ?"
	//}
	//
	//err := db.QueryRow(sqlString, id).Scan(&dbClass.Id, &dbClass.Mask, &dbClass.PowerType, &dbClass.Name)
	//switch {
	//case err == sql.ErrNoRows:
	//	return &ToonClass{}, err
	//}
	//return &dbClass, nil
}
