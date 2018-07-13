package models

// Database table model
type Toon struct {
	Id     int64
	Name   string
	Race   int64
	Class  int64
	Gender int64
	Realm  string
	Region string
}

// Create a new Toon struct
func NewToon(id int64, name string, race int64, class int64, gender int64, realm string, region string) *Toon {
	return &Toon{
		Id:     id,
		Name:   name,
		Race:   race,
		Class:  class,
		Gender: gender,
		Realm:  realm,
		Region: region,
	}
}

// Get a Toon from the database based on Id
func (db *WowDB) GetToonById(id int64) (*Toon, error) {
	var toon Toon
	err := db.QueryRow("SELECT id, name, race_id, class_id, gender, realm, region from toon where id = $1", id).Scan(&toon.Id, &toon.Name, &toon.Race, &toon.Class, &toon.Gender, &toon.Realm, &toon.Region)
	if err != nil {
		return &Toon{}, err
	}
	return &toon, nil
}

// Get all Toons from the database
func (db *WowDB) GetAllToons() []Toon {
	var toons []Toon
	rows, _ := db.Query("SELECT id, name, race_id, class_id, gender, realm, region FROM toon")
	defer rows.Close()

	for rows.Next() {
		var t Toon
		rows.Scan(&t.Id, &t.Name, &t.Race, &t.Class, &t.Gender, &t.Realm, &t.Region)
		toons = append(toons, t)
	}
	return toons
}

// Insert a new Toon into the database. Does not need an ID as the database should handle entering it.
func (db *WowDB) InsertToon(toon *Toon) error {
	_, err := db.Exec("INSERT INTO toon (name, gender, class_id, race_id, realm, region) VALUES ($1, $2, $3, $4, $5, $6)", toon.Name, toon.Gender, toon.Class, toon.Race, toon.Realm, toon.Region)
	if err != nil {
		return err
	}
	return nil
}
