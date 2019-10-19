package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/adrg/xdg"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/tabwriter"
	"time"
)

/*
 Adds "interesting" stats about WoW characters to a database. This is used to track stats over time as opposed
 to the Armory which is at this point in time.

 To add a new character, run by hand using the --add parameter

 You will need a Blizzard API key available at https://dev.battle.net/
*/

// Command line options
var opts struct {
	Add          bool `long:"add" description:"Add toon"`
	Update       bool `long:"update" description:"Update Blizzard databases"`
	Summary      bool `long:"summary" description:"Show level and ilevel for each toon"`
	EmailSummary bool `long:"emailsummary" description:"Show level and ilevel for each toon"`
	Quiet        bool `long:"quiet" description:"Do not print output"`
}

type EmailConfig struct {
	FromAddress string
	ToAddress   []string
	Server      string
}

type Config struct {
	DbDriver     string
	DbUrl        string
	ApiKey       string
	ArchiveStats bool
	ArchiveDir   string
	ClientId     string
	ClientSecret string
	LogLevel     string
	Email        EmailConfig
}

type Env struct {
	db     Datastore
	config Config
}

func main() {
	log.Trace("Starting application")
	_, err := flags.Parse(&opts)
	if err != nil {
		log.Panic(err)
	}

	_, err = xdg.ConfigFile("wowstats/wowstats.yml")
	if err != nil {
		log.Fatalf("Could not create config path")
	}

	viper.SetConfigName("wowstats")
	for _, c := range xdg.ConfigDirs {
		viper.AddConfigPath(filepath.Join(c, "wowstats"))
	}

	//viper.AddConfigPath(filepath.Join(homeDir, ".wowstats"))
	viper.AddConfigPath(".")
	viper.SetDefault("archiveDir", filepath.Join(xdg.DataHome, "wowstats", "json"))
	viper.SetDefault("archiveStats", true)

	err = viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	if !viper.IsSet("clientId") {
		log.Fatalf("Must supply clientId parameter in configuration file")
	}

	if !viper.IsSet("clientSecret") {
		log.Fatalf("Must supply clientSecret parameter in configuration file")
	}

	if !viper.IsSet("dbUrl") {
		log.Fatalf("Must supply dbUrl parameter in configuration file")
	}

	if !viper.IsSet("dbDriver") {
		log.Fatalf("Must supply dbVendor parameter in configuration file")
	}

	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		log.Fatalf("Unable to parse configuration: %v", err)
	}

	if !((config.DbDriver == "postgres") || (config.DbDriver == "mysql")) {
		log.Fatalf("Allowed database driver values are postgres or mysql")
	}

	if viper.IsSet("logLevel") {
		var logLevel, err = log.ParseLevel(config.LogLevel)
		if err != nil {
			log.SetLevel(logLevel)
		}
	}

	db, err := NewDB(config.DbDriver, config.DbUrl)
	defer db.Close()

	env := &Env{db: db, config: config}
	blizzard, err := NewBlizzard(config.ClientId, config.ClientSecret)

	if err != nil {
		log.Fatalf("Error in blizzard configuration: %v", err)
	}

	doDatabaseMigrations(db, env, blizzard)

	if opts.Update {
		log.Println("Updating info from Blizzard, please wait...")
		err = UpdateClassesFromBlizzard(env, blizzard)
		if err != nil {
			log.Fatalf("Could not update classes from Blizzard: %v", err)
		}
		err = UpdateRacesFromBlizzard(env, blizzard)
		if err != nil {
			log.Fatalf("Could not update races from Blizzard: %v", err)
		}
		log.Println("Done. Exiting.")
		os.Exit(0)
	}

	if opts.Add {
		AddToon(env, blizzard)
		os.Exit(0)
	}

	if opts.Summary {
		stats, err := env.db.GetAllToonLatestQuickSummary()
		if err != nil {
			log.Error(err)
			os.Exit(1)
		}
		w := tabwriter.NewWriter(os.Stdout, 5, 0, 3, ' ', tabwriter.AlignRight)
		_, _ = fmt.Fprintln(w, "Name\tLevel\tItem Level\tLast Modified\tDate\t")
		for _, s := range stats {
			_, _ = fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t\n", s.Toon.Name, s.Level, s.ItemLevel, s.LastModifiedAsDateTime(), s.CreatedAt.Format("2006-01-02"))
		}
		_ = w.Flush()
		os.Exit(0)
	}

	if opts.EmailSummary {
		err = DoEmailSummary(env)
		if err != nil {
			log.Printf("Error sending email: %v\n", err)
		}
		os.Exit(0)
	}

	// OK, we're going to do the normal get the stats function, we can fork these off to separate processes since
	// they aren't dependent on each other and the database will handle its own locking.

	var wg sync.WaitGroup

	toons := env.db.GetAllToons()
	for _, t := range toons {
		wg.Add(1)
		go GetAndInsertToonStats(t, env, blizzard, &wg)
	}
	wg.Wait()
	log.Trace("Exiting.")
}

// Do database migrations.
// Add additional changes after the AutoMigrate for things that AutoMigrate won't handle.
func doDatabaseMigrations(db *WowDB, env *Env, blizzard Blizzard) {

	if ! db.HasTable(&Race{}) {
		log.Debug("Migrating race")
		db.AutoMigrate(&Race{})
		err := UpdateRacesFromBlizzard(env, blizzard)
		if err != nil {
			log.Errorf("Could not update races: %v", err)
		}
	}

	if ! db.HasTable(&ToonClass{}) {
		log.Println("Migrating classes")
		db.AutoMigrate(&ToonClass{})
		err := UpdateClassesFromBlizzard(env, blizzard)
		if err != nil {
			log.Errorf("Could not update classes: %v", err)
		}
	}

	db.AutoMigrate(&Stat{})
	db.AutoMigrate(&Toon{})

	if ! db.HasTable(&ClassColor{}) {
		db.AutoMigrate(&ClassColor{})
		db.Model(&ClassColor{}).AddForeignKey("toon_class_id", "toon_classes(id)", "RESTRICT", "RESTRICT")
		var tColor = ClassColor{ToonClassID: 1, Color: "#C79C63"}
		db.Create(&tColor)
		tColor.ID = 0
		tColor.ToonClassID = 2
		tColor.Color = "#F58CBA"
		db.Create(&tColor)
		tColor.ID = 0
		tColor.ToonClassID = 3
		tColor.Color = "#ABD473"
		db.Create(&tColor)
		tColor.ID = 0
		tColor.ToonClassID = 4
		tColor.Color = "#FFF569"
		db.Create(&tColor)
		tColor.ID = 0
		tColor.ToonClassID = 5
		tColor.Color = "#F0EBE0"
		db.Create(&tColor)
		tColor.ID = 0
		tColor.ToonClassID = 6
		tColor.Color = "#C41F3B"
		db.Create(&tColor)
		tColor.ID = 0
		tColor.ToonClassID = 7
		tColor.Color = "#0070DE"
		db.Create(&tColor)
		tColor.ID = 0
		tColor.ToonClassID = 8
		tColor.Color = "#69CCF0"
		db.Create(&tColor)
		tColor.ID = 0
		tColor.ToonClassID = 9
		tColor.Color = "#9482C9"
		db.Create(&tColor)
		tColor.ID = 0
		tColor.ToonClassID = 10
		tColor.Color = "#00FF96"
		db.Create(&tColor)
		tColor.ID = 0
		tColor.ToonClassID = 11
		tColor.Color = "#FF7D0A"
		db.Create(&tColor)
		tColor.ID = 0
		tColor.ToonClassID = 12
		tColor.Color = "#A330C9"
		db.Create(&tColor)
	}

	db.Model(&Toon{}).AddForeignKey("race_id", "races(id)", "RESTRICT", "RESTRICT")
	db.Model(&Toon{}).AddForeignKey("class_id", "toon_classes(id)", "RESTRICT", "RESTRICT")
	db.Model(&Stat{}).AddForeignKey("toon_id", "toons(id)", "RESTRICT", "RESTRICT")
	db.Model(&Stat{}).AddUniqueIndex("idx_toon_id_create_date", "toon_id", "insert_date")
}

// Gets the latest stats for the specified Toon and will then save to the database.
func GetAndInsertToonStats(t Toon, env *Env, blizzard Blizzard, wg *sync.WaitGroup) {
	defer wg.Done()

	myJson, err := blizzard.GetToonJson(t)
	if err != nil {
		log.Printf("Could not get stats for %s: %v\n", t.Name, err)
		return
	}

	stats := ParseStatsFromJson(myJson)
	err = env.db.InsertStats(&stats)
	if err != nil {
		log.Printf("Error inserting stats for %v: %v\n", t.Name, err)
		return
	} else {
		if !opts.Quiet {
			log.Printf("Inserted record for %v: Level: [%v] Ilevel: [%v]", t.Name, stats.Level, stats.ItemLevel)
		}
	}

	if env.config.ArchiveStats {

		dir := filepath.Join(env.config.ArchiveDir, fmt.Sprintf("%s-%s", t.Name, t.Realm))
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			log.Printf("Could not create directory %s: %v\n", dir, err)
			// May as well return now since we can't write to the directory
			return
		}

		currentTime := time.Now().Local().Format("2006-01-02")
		fileName := filepath.Join(dir, fmt.Sprintf("%s-%s-%s.json.gz", t.Name, t.Realm, currentTime))

		// This is just for myself in the off chance I ever want to look at it, pretty print the JSON
		var pretty bytes.Buffer
		_ = json.Indent(&pretty, []byte(myJson), "", "  ")

		// Now we save it off as a gzip file
		var gzipBuffer bytes.Buffer
		var gzipWriter = gzip.NewWriter(&gzipBuffer)
		_, err = gzipWriter.Write(pretty.Bytes())
		if err != nil {
			log.Printf("Could not write JSON to buffer: %v\n", err)
		}
		err = gzipWriter.Close()
		if err != nil {
			log.Printf("Could not close gzip writer: %v\n", err)
		}

		err = ioutil.WriteFile(fileName, gzipBuffer.Bytes(), 0644)
		if err != nil {
			log.Printf("Could not write file %s: %v\n", fileName, err)
		}
	}
}

func ParseStatsFromJson(myJson string) Stat {
	var stats = new(Stat)
	//stats.ToonID = toon.ID
	stats.Level = gjson.Get(myJson, "level").Int()
	stats.AchievementPoints = gjson.Get(myJson, "achievementPoints").Int()
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
	stats.InsertDate = time.Now()
	return *stats
}

// Query the user for character to info to add to the database.
func AddToon(env *Env, blizzard Blizzard) {
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
	toon := NewToon(name, 0, 0, 0, realm, region)
	err := blizzard.GetToon(toon)
	if err != nil {
		fmt.Printf("Could not find character: %v\n", err)
		os.Exit(0)
	}

	dbClass, err := env.db.GetToonClassById(toon.ClassID)
	if err != nil {
		fmt.Printf("Could not get class info from database: %v\n", err)
		os.Exit(0)
	}

	dbRace, err := env.db.GetRaceById(toon.RaceID)
	if err != nil {
		fmt.Printf("Could not get race info from database: %v\n", err)
		os.Exit(0)
	}

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
		var dbToon Toon
		dbToon.ToonClass = *dbClass
		dbToon.Race = *dbRace
		dbToon.Name = name
		dbToon.Gender = toon.Gender
		dbToon.Realm = toon.Realm
		dbToon.Region = toon.Region

		err = env.db.InsertToon(&dbToon)
		if err != nil {
			fmt.Printf("Could not insert toon into database: %v\n", err)
			os.Exit(0)
		}
	}
}

// Update the player classes from Blizzard. This will use the API to get the classes and add them to the database. This
//probably isn't really needed, it's happened exactly twice ever, but you never know.
func UpdateClassesFromBlizzard(env *Env, blizzard Blizzard) error {
	log.Trace("Entering UpdateClassesFromBlizzard")

	classes, err := blizzard.GetClasses()
	if err != nil {
		return err
	}

	for _, c := range classes {
		toonClass, err := env.db.GetToonClassById(c.ID)
		//log.Printf("Back from get by id, err: %v\n", err)
		if err != nil {
			//log.Printf("Adding class: %v\n", c.Name)
			toonClass.Name = c.Name
			toonClass.ID = c.ID
			toonClass.PowerType = c.PowerType
			toonClass.Mask = c.Mask
			err = env.db.InsertToonClass(toonClass)
			if err != nil {
				log.Printf("Could not insert class %s: %v\n", c.Name, err)
			}
		}
	}
	log.Trace("Exiting UpdateClassesFromBlizzard")
	return nil
}

// Update the database with the data from Blizzard. We force insert the ID here and use the same ID as
// Blizzard so that we can map things correctly.
func UpdateRacesFromBlizzard(env *Env, blizzard Blizzard) error {
	log.Trace("Entering UpdateRacesFromBlizzard")
	races, err := blizzard.GetRaces()

	if err != nil {
		return err
	}

	for _, r := range races {
		//log.Printf("Searching for race id: %v\n", r.ID)
		race, err := env.db.GetRaceById(r.ID)
		//log.Printf("Back from get by id, err: %v\n", err)
		if err != nil {
			//log.Printf("Inserting race: [%v]\n", r.Name)
			race.ID = r.ID
			race.Name = r.Name
			race.Mask = r.Mask
			race.Side = r.Side
			err = env.db.InsertRace(race)
			if err != nil {
				log.Printf("Could not insert race %s: %v\n", r.Name, err)
			}
		}
	}
	log.Trace("Exiting UpdateRacesFromBlizzard")
	return nil
}
