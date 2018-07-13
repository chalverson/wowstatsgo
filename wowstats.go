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
	"github.com/tidwall/gjson"
	"gopkg.in/resty.v1"
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

	if opts.Update {
		log.Println("Updating info from Blizzard, please wait...")
		UpdateClassesFromBlizzard(env)
		UpdateRacesFromBlizzard(env)
		log.Println("Done. Exiting.")
		os.Exit(0)
	}

	if opts.Add {
		AddToon(env)
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
		DoEmailSummary(env)
		os.Exit(0)
	}

	// OK, we're going to do the normal get the stats function, we can fork these off to separate processes since
	// they aren't dependent on each other and the database will handle its own locking.

	var wg sync.WaitGroup

	toons := env.db.GetAllToons()
	for _, t := range toons {
		wg.Add(1)
		go GetAndInsertToonStats(t, env, &wg)
	}
	wg.Wait()
}

// Gets the latest stats for the specified Toon and will then save to the database.
func GetAndInsertToonStats(t models.Toon, env *Env, wg *sync.WaitGroup) {
	defer wg.Done()
	currentTime := time.Now().Local().Format("2006-01-02")

	url := fmt.Sprintf("https://%s.api.battle.net/wow/character/%s/%s", t.Region, t.Realm, t.Name)
	resp, err := resty.R().SetQueryParams(map[string]string{
		"fields": "statistics,items,pets,mounts",
		"apikey": env.config.ApiKey,
	}).SetHeader("Accept", "application/json").Get(url)
	if err != nil {
		log.Println("Could not get things from Blizzard")
	}

	myJson := resp.String()
	var stats = new(models.Stats)
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

	err = env.db.InsertStats(stats)
	if err != nil {
		log.Printf("Error inserting stats for %v: %v\n", t.Name, err)
		return
	} else {
		if !opts.Quiet {
			log.Printf("Inserted record for %v: Level: [%v] Ilevel: [%v]", t.Name, stats.Level, stats.ItemLevel)
		}
	}

	dir := filepath.Join(env.config.ArchiveDir, fmt.Sprintf("%s-%s", t.Name, t.Realm))
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

// Query the user for character to info to add to the database.
func AddToon(env *Env) {
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
	apiKey := env.config.ApiKey
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

	dbClass, err := env.db.GetToonClassById(toonClass)
	dbRace, err := env.db.GetRaceById(toonRace)

	toon := models.NewToon(0, name, toonRace, toonClass, toonGender, realm, region)

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
		env.db.InsertToon(toon)
	}
}

// Update the player classes from Blizzard. This will use the API to get the classes and add them to the database. This
//probably isn't really needed, it's happened exactly twice ever, but you never know.
func UpdateClassesFromBlizzard(env *Env) {
	resp, err := resty.R().SetQueryParams(map[string]string{
		"apikey": env.config.ApiKey,
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

		toonClass, err := env.db.GetToonClassById(id)
		if err != nil {
			log.Printf("Adding class: %v\n", name)
			toonClass.Name = name
			toonClass.Id = id
			toonClass.PowerType = powerType
			toonClass.Mask = mask
			err = env.db.InsertToonClass(toonClass)
			if err != nil {
				log.Printf("Could not insert class %s: %v", name, err)
			}
		}
	}
}

// Update the database with the data from Blizzard. We force insert the ID here and use the same ID as
// Blizzard so that we can map things correctly.
func UpdateRacesFromBlizzard(env *Env) {
	resp, err := resty.R().SetQueryParams(map[string]string{
		"apikey": env.config.ApiKey,
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

		race, err := env.db.GetRaceById(id)
		if err != nil {
			race.Id = id
			race.Name = name
			race.Mask = mask
			race.Side = side
			env.db.InsertRace(race)
		}
	}
}
