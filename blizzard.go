package main

import (
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"gopkg.in/resty.v1"
)

// Functions to interact with Blizzard.
type Blizzard interface {
	GetToonJson(toon Toon) (string, error)
	GetClasses() ([]ToonClass, error)
	GetRaces() ([]Race, error)
	GetToon(toon *ToonDto) error
}

// Configuration information for interacting with Blizzard.
type BlizzardHttp struct {
	ClientId     string
	ClientSecret string
	AccessToken  string
}

func NewBlizzard(clientId string, clientSecret string) (*BlizzardHttp, error) {
	url := "https://us.battle.net/oauth/token"
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

func (blizzard *BlizzardHttp) GetToon(toon *ToonDto) error {
	url := fmt.Sprintf("https://%s.api.blizzard.com/wow/character/%s/%s", toon.Region, toon.Realm, toon.Name)

	resp, err := resty.R().SetAuthToken(blizzard.AccessToken).SetHeader("Accept", "application/json").Get(url)

	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return errors.New("could not find character, status code not 200")
	}

	body := resp.String()

	toon.ClassID = gjson.Get(body, "class").Int()
	toon.RaceID = gjson.Get(body, "race").Int()
	toon.Gender = gjson.Get(body, "gender").Int()
	toon.Name = gjson.Get(body, "name").String()

	return nil
}

func (blizzard *BlizzardHttp) GetClasses() ([]ToonClass, error) {
	resp, err := resty.R().SetAuthToken(blizzard.AccessToken).SetHeader("Accept", "application/json").Get("https://us.api.blizzard.com/wow/data/character/classes")
	if err != nil {
		return nil, err
	}
	respJson := resp.String()
	result := gjson.Get(respJson, "classes")
	var classes []ToonClass
	for _, r := range result.Array() {
		id := gjson.Get(r.String(), "id").Int()
		mask := gjson.Get(r.String(), "mask").Int()
		powerType := gjson.Get(r.String(), "powerType").String()
		name := gjson.Get(r.String(), "name").String()
		tmpClass := ToonClass{
			ID:        id,
			Mask:      mask,
			PowerType: powerType,
			Name:      name,
		}
		classes = append(classes, tmpClass)
	}
	return classes, nil
}

func (blizzard *BlizzardHttp) GetToonJson(toon Toon) (string, error) {
	url := fmt.Sprintf("https://%s.api.blizzard.com/wow/character/%s/%s", toon.Region, toon.Realm, toon.Name)
	resp, err := resty.R().SetQueryParams(map[string]string{
		"fields": "statistics,items,pets,mounts",
	}).SetAuthToken(blizzard.AccessToken).SetHeader("Accept", "application/json").Get(url)
	if err != nil {
		return "", err
	}

	myJson := resp.String()
	return myJson, nil
}

func (blizzard *BlizzardHttp) GetRaces() ([]Race, error) {
	resp, err := resty.R().SetAuthToken(blizzard.AccessToken).SetHeader("Accept", "application/json").Get("https://us.api.blizzard.com/wow/data/character/races")
	if err != nil {
		return nil, err
	}

	respJson := resp.String()
	result := gjson.Get(respJson, "races")
	var races []Race

	for _, r := range result.Array() {
		id := gjson.Get(r.String(), "id").Int()
		mask := gjson.Get(r.String(), "mask").Int()
		side := gjson.Get(r.String(), "side").String()
		name := gjson.Get(r.String(), "name").String()

		race := Race{
			ID:   id,
			Name: name,
			Mask: mask,
			Side: side,
		}
		races = append(races, race)
	}

	return races, nil
}

