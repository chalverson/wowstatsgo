package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/jessevdk/go-flags"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"text/tabwriter"
	"time"
	"github.com/chalverson/wowstatsgo/models"
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
	DbUrl      string
	ApiKey     string
	ArchiveDir string
	Email      EmailConfig
}

type Env struct {
	db     models.Datastore
	config Config
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

	if !viper.IsSet("apiKey") {
		log.Fatalf("Must supply apiKey parameter in configuration file")
	}

	if !viper.IsSet("dbUrl") {
		log.Fatalf("Must supply dbUrl parameter in configuration file")
	}

	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		log.Fatalf("Unable to parse configuration: %v", err)
	}

	db, err := models.NewDB(config.DbUrl)
	defer db.Close()
	env := &Env{db: db, config: config}
	blizzard := &BlizzardHttp{ApiKey: config.ApiKey}

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
		stats := env.db.GetAllToonLatestQuickSummary()
		w := tabwriter.NewWriter(os.Stdout, 5, 0, 3, ' ', tabwriter.AlignRight)
		fmt.Fprintln(w, "Name\tLevel\tItem Level\tLast Modified\tDate\t")
		for _, s := range stats {
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t\n", s.Toon.Name, s.Level, s.ItemLevel, s.LastModifiedAsDateTime(), s.CreateDate.Format("2006-01-02"))
		}
		w.Flush()
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
}

// Gets the latest stats for the specified Toon and will then save to the database.
func GetAndInsertToonStats(t models.Toon, env *Env, blizzard Blizzard, wg *sync.WaitGroup) {
	defer wg.Done()

	stats, myJson, err := blizzard.GetToonStats(t)
	if err != nil {
		log.Printf("Could not get stats for %s: %v\n", t.Name, err)
		return
	}

	err = env.db.InsertStats(&stats)
	if err != nil {
		log.Printf("Error inserting stats for %v: %v\n", t.Name, err)
		return
	} else {
		if !opts.Quiet {
			log.Printf("Inserted record for %v: Level: [%v] Ilevel: [%v]", t.Name, stats.Level, stats.ItemLevel)
		}
	}

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
	json.Indent(&pretty, []byte(myJson), "", "  ")

	// Now we save it off as a gzip file
	var gzipBuffer bytes.Buffer
	var gzipWriter = gzip.NewWriter(&gzipBuffer)
	_, err = gzipWriter.Write(pretty.Bytes())
	if err != nil {
		log.Printf("Could not write JSON to buffer: %v\n", err)
	}
	gzipWriter.Close()

	err = ioutil.WriteFile(fileName, gzipBuffer.Bytes(), 0644)
	if err != nil {
		log.Printf("Could not write file %s: %v\n", fileName, err)
	}
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
	toon := models.NewToon(0, name, 0, 0, 0, realm, region)
	err := blizzard.GetToon(toon)
	if err != nil {
		fmt.Printf("Could not find character: %v\n", err)
		os.Exit(0)
	}

	dbClass, err := env.db.GetToonClassById(toon.Class)
	if err != nil {
		fmt.Printf("Could not get class info from database: %v\n", err)
		os.Exit(0)
	}

	dbRace, err := env.db.GetRaceById(toon.Race)
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
		err = env.db.InsertToon(toon)
		if err != nil {
			fmt.Printf("Could not insert toon into database: %v\n", err)
			os.Exit(0)
		}
	}
}

// Update the player classes from Blizzard. This will use the API to get the classes and add them to the database. This
//probably isn't really needed, it's happened exactly twice ever, but you never know.
func UpdateClassesFromBlizzard(env *Env, blizzard Blizzard) error {
	classes, err := blizzard.GetClasses()
	if err != nil {
		return err
	}

	for _, c := range classes {
		toonClass, err := env.db.GetToonClassById(c.Id)
		if err != nil {
			log.Printf("Adding class: %v\n", c.Name)
			toonClass.Name = c.Name
			toonClass.Id = c.Id
			toonClass.PowerType = c.PowerType
			toonClass.Mask = c.Mask
			err = env.db.InsertToonClass(toonClass)
			if err != nil {
				log.Printf("Could not insert class %s: %v\n", c.Name, err)
			}
		}
	}
	return nil
}

// Update the database with the data from Blizzard. We force insert the ID here and use the same ID as
// Blizzard so that we can map things correctly.
func UpdateRacesFromBlizzard(env *Env, blizzard Blizzard) error {
	races, err := blizzard.GetRaces()

	if err != nil {
		return err
	}

	for _, r := range races {
		race, err := env.db.GetRaceById(r.Id)
		if err != nil {
			race.Id = r.Id
			race.Name = r.Name
			race.Mask = r.Mask
			race.Side = r.Side
			err = env.db.InsertRace(race)
			if err != nil {
				log.Printf("Could not insert race %s: %v\n", r.Name, err)
			}
		}
	}
	return nil
}
