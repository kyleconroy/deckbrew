package main

import (
	"encoding/json"
	"io/ioutil"
)

type MTGCollection map[string]MTGSet

type MTGSet struct {
	Name     string    `json:"name"`
	Code     string    `json:"code"`
	Released string    `json:"releaseDate"`
	Border   string    `json:"border"`
	Type     string    `json:"type"`
	Cards    []MTGCard `json:"cards"`
}

type MTGCard struct {
	Artist        string      `json:"artist"`
	Border        string      `json:"border"`
	Colors        []string    `json:"colors"`
	ConvertedCost float64     `json:"cmc"`
	Flavor        string      `json:"flavor"`
	HandModifier  int         `json:"hand"`
	Layout        string      `json:"layout"`
	LifeModifier  int         `json:"life"`
	Loyalty       int         `json:"loyalty"`
	ManaCost      string      `json:"manaCost"`
	MultiverseId  int         `json:"multiverseid"`
	Name          string      `json:"name"`
	Names         []string    `json:"names"`
	Number        string      `json:"number"`
	Power         string      `json:"power"`
	Rarity        string      `json:"rarity"`
	Rulings       []MTGRuling `json:"rulings"`
	Subtypes      []string    `json:"subtypes"`
	Supertypes    []string    `json:"supertypes"`
	Text          string      `json:"text"`
	Toughness     string      `json:"toughness"`
	Type          string      `json:"type"`
	Types         []string    `json:"types"`
	Watermark     string      `json:"watermark"`
}

type MTGRuling struct {
	Date string `json:"date"`
	Text string `json:"text"`
}

type MTGFormat struct {
	Name       string   `json:"name"`
	Sets       []string `json:"sets"`
	Banned     []Card   `json:"banned"`
	Restricted []Card   `json:"restricted"`
}

func LoadFormat(path string) (MTGFormat, error) {
	blob, err := ioutil.ReadFile(path)

	if err != nil {
		return MTGFormat{}, err
	}

	var format MTGFormat
	err = json.Unmarshal(blob, &format)

	if err != nil {
		return MTGFormat{}, err
	}
	return format, nil
}

func LoadCollection(path string) (MTGCollection, error) {
	blob, err := ioutil.ReadFile(path)

	if err != nil {
		return MTGCollection{}, err
	}

	var collection MTGCollection
	err = json.Unmarshal(blob, &collection)
	return collection, err
}
