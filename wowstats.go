package main

import (
	_ "github.com/lib/pq"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"time"
	"fmt"
	"gopkg.in/resty.v1"
	"github.com/tidwall/gjson"
	"bytes"
	"encoding/json"
	"compress/gzip"
	"io/ioutil"
	"database/sql"
	"bufio"
	"strings"
	"text/tabwriter"
	"os/user"
	"path/filepath"
	"net/smtp"
	"html/template"
	"github.com/spf13/viper"
	"sync"
)

/*
 Adds "interesting" stats about WoW characters to a database. This is used to track stats over time as opposed
 to the Armory which is at this point in time.

 To add a new character, run by hand using the --add parameter

 You will need a Blizzard API key available at https://dev.battle.net/
*/

// Command line options
var opts struct {
	Config       bool `long:"config" description:"Run configuration"`
	Add          bool `long:"add" description:"Add toon"`
	Update       bool `long:"update" description:"Update Blizzard databases"`
	Summary      bool `long:"summary" description:"Show level and ilevel for each toon"`
	EmailSummary bool `long:"emailsummary" description:"Show level and ilevel for each toon"`
	Quiet        bool `long:"quiet" description:"Do not print output"`
}

type WowDB struct {
	*sql.DB
}

func NewPostgresDB(connStr string) *WowDB {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}
	return &WowDB{db}
}

type Race struct {
	Id   int64
	Mask int64
	Side string
	Name string
}

type ToonClass struct {
	Id        int64
	Mask      int64
	PowerType string
	Name      string
}

// Structure to hold info for a WoW Toon
type Toon struct {
	Id     int64
	Name   string
	Race   int64
	Class  int64
	Gender int64
	Realm  string
	Region string
}

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

type ToonDto struct {
	Id     int64
	Name   string
	Race   *Race
	Class  *ToonClass
	Gender string
	Realm  string
	Region string
}

// Map the stats table
type Stats struct {
	Toon             *Toon
	LastModified     int64
	CreateDate       time.Time
	Level            int64
	AchievementPoint int64
	ExaltedReps      int64
	MountsCollected  int64
	QuestsCompleted  int64
	FishCaught       int64
	PetsCollected    int64
	PetBattlesWon    int64
	PetBattlesPvpWon int64
	ItemLevel        int64
	HonorableKills   int64
}

type EmailRequest struct {
	from    string
	to      []string
	subject string
	body    string
	server  string
}

func NewEmailRequest(to []string, from string, subject, server, body string) *EmailRequest {
	return &EmailRequest{
		to:      to,
		from:    from,
		subject: subject,
		body:    body,
		server:  server,
	}
}

func (r *EmailRequest) SendEmail() (bool, error) {
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	subject := "Subject: " + r.subject + "\n"
	addr := r.server
	emailFrom := "cdh@halverson.org"

	c, err1 := smtp.Dial(addr)
	if err1 != nil {
		log.Fatal(err1)
	}
	defer c.Close()
	c.Mail(emailFrom)
	for _, recipient := range r.to {
		c.Rcpt(recipient)
	}
	wc, err := c.Data()
	if err != nil {
		log.Fatal(err)
	}
	defer wc.Close()
	buf := bytes.NewBufferString(subject + mime + "\n" + r.body)
	if _, err = buf.WriteTo(wc); err != nil {
		log.Fatal(err)
	}

	return true, nil
}

func (r *EmailRequest) ParseTemplate(templateFileName string, data interface{}) error {
	t := template.New("")
	t.Funcs(template.FuncMap{"zebra": func(i int) bool { return i%2 == 0 }})
	t.Parse(templateFileName)
	buf := new(bytes.Buffer)
	if err := t.Execute(buf, data); err != nil {
		return err
	}
	r.body = buf.String()
	return nil
}

func DoEmailSummary(db *WowDB, config Config) {
	stats := db.GetAllToonLatestQuickSummary()

	const tpl = `
<table border="0" cellspacing="0" cellpadding="5">
        <caption>WoW Stats</caption>
    <thead>
    <tr><th>Name</th><th>Level</th><th>Item Level</th><th>Date</th></tr>
    </thead>
    <tbody>
{{range $idx, $b := .}}
{{if zebra $idx}}<tr bgcolor="#C4C2C2">{{else}}<tr bgcolor="#DBDBDB">{{end}}
<td>{{$b.Toon.Name}}</td><td>{{$b.Level}}</td><td>{{$b.ItemLevel}}</td><td>{{$b.CreateDate.Format "2006-01-02"}}</td></tr>
{{end}}
</tbody></table><p>
`
	r := NewEmailRequest(config.Email.ToAddress, config.Email.FromAddress, "WoW Stats", config.Email.Server, "")
	err := r.ParseTemplate(tpl, stats)
	if err != nil {
		fmt.Println("Err in parsing template", err)
		os.Exit(0)
	}
	_, err = r.SendEmail()
	if err != nil {
		fmt.Println("Mail failed", err)
	}

}

type EmailConfig struct {
	FromAddress string
	ToAddress   []string
	Server      string
}

type Config struct {
	DbUrl      string
	ApiKey     string
	ArchiveDir string
	Email      EmailConfig
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		log.Panic(err)
	}

	usr, _ := user.Current()
	homeDir := usr.HomeDir

	os.MkdirAll(filepath.Join(homeDir, ".wowstats"), 0755)
	viper.SetConfigName("wowstats")
	viper.AddConfigPath("$HOME/.wowstats")
	viper.AddConfigPath(filepath.Join(homeDir, ".wowstats"))
	viper.AddConfigPath(".")
	viper.SetDefault("archiveDir", filepath.Join(homeDir, ".wowstats", "json"))

	err = viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	if ! viper.IsSet("apiKey") {
		log.Fatalf("Must supply apiKey parameter in configuration file")
	}

	if ! viper.IsSet("dbUrl") {
		log.Fatalf("Must supply dbUrl parameter in configuration file")
	}

	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		log.Fatalf("Unable to parse configuration: %v", err)
	}

	db := NewPostgresDB(config.DbUrl)
	defer db.Close()

	if opts.Update {
		log.Println("Updating info from Blizzard, please wait...")
		UpdateClassesFromBlizzard(db, &config)
		UpdateRacesFromBlizzard(db, &config)
		log.Println("Done. Exiting.")
		os.Exit(0)
	}

	if opts.Add {
		AddToon(db, &config)
		os.Exit(0)
	}

	if opts.Summary {
		stats := db.GetAllToonLatestQuickSummary()
		w := tabwriter.NewWriter(os.Stdout, 5, 0, 3, ' ', tabwriter.AlignRight)
		fmt.Fprintln(w, "Name\tLevel\tItem Level\tDate\t")
		for _, s := range stats {
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\t\n", s.Toon.Name, s.Level, s.ItemLevel, s.CreateDate.Format("2006-01-02"))
		}
		w.Flush()
		os.Exit(0)
	}

	if opts.EmailSummary {
		DoEmailSummary(db, config)
		os.Exit(0)
	}

	// OK, we're going to do the normal get the stats function, we can fork these off to separate processes since
	// they aren't dependent on each other and the database will handle its own locking.

	var wg sync.WaitGroup

	toons := db.GetAllToons()
	for _, t := range toons {
		wg.Add(1)
		go GetAndInsertToonStats(t, db, &config, &wg)
	}
	wg.Wait()
}

func GetAndInsertToonStats(t Toon, db *WowDB, config *Config, wg *sync.WaitGroup) {
	defer wg.Done()
	currentTime := time.Now().Local().Format("2006-01-02")

	url := fmt.Sprintf("https://%s.api.battle.net/wow/character/%s/%s", t.Region, t.Realm, t.Name)
	resp, err := resty.R().SetQueryParams(map[string]string{
		"fields": "statistics,items,pets,mounts",
		"apikey": config.ApiKey,
	}).SetHeader("Accept", "application/json").Get(url)
	if err != nil {
		log.Println("Could not get things from Blizzard")
	}

	myJson := resp.String()
	var stats = new(Stats)
	stats.Toon = &t
	stats.Level = gjson.Get(myJson, "level").Int()
	stats.AchievementPoint = gjson.Get(myJson, "achievementPoints").Int()
	stats.ExaltedReps = gjson.Get(myJson, "statistics.subCategories.#[id==130].subCategories.#[id==147].statistics.#[id=377].quantity").Int()
	stats.MountsCollected = gjson.Get(myJson, "mounts.numCollected").Int()
	stats.QuestsCompleted = gjson.Get(myJson, "statistics.subCategories.#[id==133].statistics.#[id=98].quantity").Int()
	stats.FishCaught = gjson.Get(myJson, "statistics.subCategories.#[id==132].subCategories.#[id==178].statistics.#[id==1518].quantity").Int()
	stats.PetsCollected = gjson.Get(myJson, "pets.numCollected").Int()
	stats.PetBattlesWon = gjson.Get(myJson, "statistics.subCategories.#[id==15219].statistics.#[id==8278].quantity").Int()
	stats.PetBattlesPvpWon = gjson.Get(myJson, "statistics.subCategories.#[id==15219].statistics.#[id==8286].quantity").Int()
	stats.ItemLevel = gjson.Get(myJson, "items.averageItemLevel").Int()
	stats.HonorableKills = gjson.Get(myJson, "totalHonorableKills").Int()
	stats.LastModified = gjson.Get(myJson, "lastModified").Int()
	stats.CreateDate = time.Now().UTC()

	err = db.InsertStats(stats)
	if err != nil {
		log.Printf("Error inserting stats for %v: %v\n", t.Name, err)
		return
	} else {
		if ! opts.Quiet {
			log.Printf("Inserted record for %v: Level: [%v] Ilevel: [%v]", t.Name, stats.Level, stats.ItemLevel)
		}
	}

	dir := filepath.Join(config.ArchiveDir, fmt.Sprintf("%s-%s", t.Name, t.Realm))
	os.MkdirAll(dir, 0755)

	fileName := filepath.Join(dir, fmt.Sprintf("%s-%s-%s.json.gz", t.Name, t.Realm, currentTime))

	// This is just for myself in the off chance I ever want to look at it, pretty print the JSON
	var pretty bytes.Buffer
	json.Indent(&pretty, []byte(myJson), "", "  ")

	// Now we save it off as a gzip file
	var gzipBuffer bytes.Buffer
	var w = gzip.NewWriter(&gzipBuffer)
	w.Write(pretty.Bytes())
	w.Close()

	ioutil.WriteFile(fileName, gzipBuffer.Bytes(), 0644)

}

func (db *WowDB) InsertStats(stats *Stats) error {
	_, err := db.Exec("INSERT INTO stats (toon_id, last_modified, create_date, level, achievement_points, number_exalted, mounts_owned, quests_completed, fish_caught, pets_owned, pet_battles_won, pet_battles_pvp_won, item_level, honorable_kills) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)",
		stats.Toon.Id, stats.LastModified, stats.CreateDate, stats.Level, stats.AchievementPoint, stats.ExaltedReps, stats.MountsCollected, stats.QuestsCompleted,
		stats.FishCaught, stats.PetsCollected, stats.PetBattlesWon, stats.PetBattlesPvpWon, stats.ItemLevel, stats.HonorableKills)

	if err != nil {
		return err
	}
	return nil
}

func AddToon(db *WowDB, config *Config) {
	fmt.Printf("Character name: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	name := strings.Title(strings.ToLower(scanner.Text()))
	fmt.Printf("Realm: ")
	scanner.Scan()
	realm := strings.Title(strings.ToLower(scanner.Text()))
	fmt.Printf("Region (us, eu, kr, tw): ")
	scanner.Scan()
	region := strings.ToLower(scanner.Text())
	fmt.Printf("Region: [%v]\n", region)

	fmt.Println("Looking up character, please wait...")
	apiKey := config.ApiKey
	url := fmt.Sprintf("https://%s.api.battle.net/wow/character/%s/%s", region, realm, name)

	resp, err := resty.R().SetQueryParams(map[string]string{
		"apikey": apiKey,
	}).SetHeader("Accept", "application/json").Get(url)

	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	if resp.StatusCode() != 200 {
		fmt.Printf("Could not find character [%v]. Exiting.\n", name)
		os.Exit(0)
	}

	body := resp.String()

	toonClass := gjson.Get(body, "class").Int()
	toonRace := gjson.Get(body, "race").Int()
	toonGender := gjson.Get(body, "gender").Int()

	dbClass, err := db.GetClassById(toonClass)
	dbRace, err := db.GetRaceById(toonRace)

	toon := NewToon(0, name, toonRace, toonClass, toonGender, realm, region)

	fmt.Println("Found character, please verify:")
	fmt.Printf("  Name:  %v\n", name)
	fmt.Printf("  Race:  %v\n", dbRace.Name)
	fmt.Printf("  Class: %v\n", dbClass.Name)
	fmt.Printf("  Realm: %v\n", realm)
	fmt.Print("\nAdd character? ")
	scanner.Scan()
	addResp := strings.ToLower(scanner.Text())
	if addResp == "y" || addResp == "" {
		fmt.Println("Adding character")
		db.InsertToon(toon)
	}
}

func (db *WowDB) InsertToon(toon *Toon) {
	_, err := db.Exec("INSERT INTO toon (name, gender, class_id, race_id, realm, region) VALUES ($1, $2, $3, $4, $5, $6)", toon.Name, toon.Gender, toon.Class, toon.Race, toon.Realm, toon.Region)
	if err != nil {
		log.Println(err)
	}
}

/*
Update the player classes from Blizzard. This will use the API to get the classes and add them to the database. This
probably isn't really needed, it's happened exactly twice ever, but you never know.
 */
func UpdateClassesFromBlizzard(db *WowDB, config *Config) {
	resp, err := resty.R().SetQueryParams(map[string]string{
		"apikey": config.ApiKey,
	}).SetHeader("Accept", "application/json").Get("https://us.api.battle.net/wow/data/character/classes")
	if err != nil {
		log.Println("Could not get classes from Blizzard")
	}
	respJson := resp.String()
	result := gjson.Get(respJson, "classes")

	// There's probably a better way
	for _, r := range result.Array() {
		id := gjson.Get(r.String(), "id").Int()
		mask := gjson.Get(r.String(), "mask").Int()
		powerType := gjson.Get(r.String(), "powerType").String()
		name := gjson.Get(r.String(), "name").String()

		toonClass, err := db.GetClassById(id)
		if err != nil {
			log.Printf("Adding class: %v\n", name)
			toonClass.Name = name
			toonClass.Id = id
			toonClass.PowerType = powerType
			toonClass.Mask = mask
			db.InsertClass(toonClass)
		}
	}
}

func (db *WowDB) InsertClass(toonClass *ToonClass) {
	_, err := db.Exec("INSERT INTO classes (id, mask, powerType, name) VALUES ($1, $2, $3, $4)", toonClass.Id, toonClass.Mask, toonClass.PowerType, toonClass.Name)
	if err != nil {
		log.Fatal("Could not create statement to insert into classes: ", err)
	}

}

// Update the database with the data from Blizzard. We force insert the ID here and use the same ID as
// Blizzard so that we can map things correctly.
func UpdateRacesFromBlizzard(db *WowDB, config *Config) {
	resp, err := resty.R().SetQueryParams(map[string]string{
		"apikey": config.ApiKey,
	}).SetHeader("Accept", "application/json").Get("https://us.api.battle.net/wow/data/character/races")
	if err != nil {
		log.Println("Could not get races from Blizzard")
	}
	respJson := resp.String()
	result := gjson.Get(respJson, "races")

	for _, r := range result.Array() {
		id := gjson.Get(r.String(), "id").Int()
		mask := gjson.Get(r.String(), "mask").Int()
		side := gjson.Get(r.String(), "side").String()
		name := gjson.Get(r.String(), "name").String()

		race, err := db.GetRaceById(id)
		if err != nil {
			race.Id = id
			race.Name = name
			race.Mask = mask
			race.Side = side
			db.InsertRace(race)
		}
	}
}

func (db *WowDB) InsertRace(race *Race) {
	_, err := db.Exec("INSERT INTO races (id, mask, side, name) VALUES ($1, $2, $3, $4)", race.Id, race.Mask, race.Side, race.Name)
	if err != nil {
		log.Fatal("Could not insert into races: ", err)
	}

}

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

func (db *WowDB) GetClassById(id int64) (*ToonClass, error) {
	var dbClass ToonClass
	err := db.QueryRow("SELECT id, mask, powerType, name FROM classes WHERE id = $1", id).Scan(&dbClass.Id, &dbClass.Mask, &dbClass.PowerType, &dbClass.Name)
	switch {
	case err == sql.ErrNoRows:
		return &ToonClass{}, err
	}
	return &dbClass, nil
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

func (db *WowDB) GetToonById(id int64) (*Toon, error) {
	var toon Toon
	err := db.QueryRow("SELECT id, name, race_id, class_id, gender, realm, region from toon where id = $1", id).Scan(&toon.Id, &toon.Name, &toon.Race, &toon.Class, &toon.Gender, &toon.Realm, &toon.Region)
	if err != nil {
		return &Toon{}, err
	}
	return &toon, nil
}

func (db *WowDB) GetAllToonLatestQuickSummary() []Stats {
	rows, _ := db.Query("select t.id, s.level, s.item_level, s.create_date, t.name from stats s join toon t on s.toon_id = t.id and s.create_date::date = (select max(create_date::date) from stats) ORDER BY s.level, s.item_level DESC, t.name ASC")
	defer rows.Close()
	var stats []Stats
	for rows.Next() {
		var id int64
		var s Stats
		var name string
		rows.Scan(&id, &s.Level, &s.ItemLevel, &s.CreateDate, &name)
		s.Toon, _ = db.GetToonById(id)
		stats = append(stats, s)
	}
	return stats
}
