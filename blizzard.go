package main

import (
	"errors"
	"fmt"
	"github.com/chalverson/wowstatsgo/models"
	"github.com/tidwall/gjson"
	"gopkg.in/resty.v1"
	"time"
)

type Blizzard interface {
	GetToonStats(toon models.Toon) (models.Stats, string, error)
	GetClasses() ([]models.ToonClass, error)
	GetRaces() ([]models.Race, error)
	GetToon(toon *models.Toon) error
}

type BlizzardHttp struct {
	ClientId     string
	ClientSecret string
	AccessToken  string
}

func NewBlizzard(clientId string, clientSecret string) (*BlizzardHttp, error) {
	url := fmt.Sprintf("https://us.battle.net/oauth/token")
	resp, err := resty.R().SetBasicAuth(clientId, clientSecret).SetQueryParam("grant_type", "client_credentials").Get(url)
	if err != nil {
		return nil, err
	}

	body := resp.String()

	if resp.StatusCode() != 200 {
		errorDescription := gjson.Get(body, "error_description").String()
		return nil, errors.New(fmt.Sprintf("Could not get auth token: %s", errorDescription))
	}

	accessToken := gjson.Get(body, "access_token").String()

	blizzardHttp := &BlizzardHttp{ClientId: clientId, ClientSecret: clientSecret, AccessToken: accessToken}
	return blizzardHttp, nil
}

func (blizzard *BlizzardHttp) GetToon(toon *models.Toon) error {
	url := fmt.Sprintf("https://%s.api.blizzard.com/wow/character/%s/%s", toon.Region, toon.Realm, toon.Name)

	resp, err := resty.R().SetAuthToken(blizzard.AccessToken).SetHeader("Accept", "application/json").Get(url)

	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return errors.New("could not find character, status code not 200")
	}

	body := resp.String()

	toon.Class = gjson.Get(body, "class").Int()
	toon.Race = gjson.Get(body, "race").Int()
	toon.Gender = gjson.Get(body, "gender").Int()
	toon.Name = gjson.Get(body, "name").String()

	return nil
}

func (blizzard *BlizzardHttp) GetClasses() ([]models.ToonClass, error) {
	resp, err := resty.R().SetAuthToken(blizzard.AccessToken).SetHeader("Accept", "application/json").Get("https://us.api.blizzard.com/wow/data/character/classes")
	if err != nil {
		return nil, err
	}
	respJson := resp.String()
	result := gjson.Get(respJson, "classes")
	var classes []models.ToonClass
	for _, r := range result.Array() {
		id := gjson.Get(r.String(), "id").Int()
		mask := gjson.Get(r.String(), "mask").Int()
		powerType := gjson.Get(r.String(), "powerType").String()
		name := gjson.Get(r.String(), "name").String()
		tmpClass := models.ToonClass{
			Id:        id,
			Mask:      mask,
			PowerType: powerType,
			Name:      name,
		}
		classes = append(classes, tmpClass)
	}
	return classes, nil
}

func (blizzard *BlizzardHttp) GetToonStats(toon models.Toon) (models.Stats, string, error) {
	url := fmt.Sprintf("https://%s.api.blizzard.com/wow/character/%s/%s", toon.Region, toon.Realm, toon.Name)
	resp, err := resty.R().SetQueryParams(map[string]string{
		"fields": "statistics,items,pets,mounts",
	}).SetAuthToken(blizzard.AccessToken).SetHeader("Accept", "application/json").Get(url)
	if err != nil {
		return models.Stats{}, "", err
	}

	myJson := resp.String()
	var stats = new(models.Stats)
	stats.Toon = &toon
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

	return *stats, myJson, nil
}

func (blizzard *BlizzardHttp) GetRaces() ([]models.Race, error) {
	resp, err := resty.R().SetAuthToken(blizzard.AccessToken).SetHeader("Accept", "application/json").Get("https://us.api.blizzard.com/wow/data/character/races")
	if err != nil {
		return nil, err
	}

	respJson := resp.String()
	result := gjson.Get(respJson, "races")
	var races []models.Race

	for _, r := range result.Array() {
		id := gjson.Get(r.String(), "id").Int()
		mask := gjson.Get(r.String(), "mask").Int()
		side := gjson.Get(r.String(), "side").String()
		name := gjson.Get(r.String(), "name").String()

		race := models.Race{
			Id:   id,
			Name: name,
			Mask: mask,
			Side: side,
		}
		races = append(races, race)
	}

	return races, nil
}
