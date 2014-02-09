package main

import (
	"fmt"
	"net/url"
	"strconv"
)

type Search struct {
	Query map[string]interface{}
	Args  url.Values
}

func (s *Search) extractStrings(searchTerm, key string, allowed map[string]bool) error {
	items := s.Args[searchTerm]

	if len(items) == 0 {
		return nil
	}

	for _, t := range items {
		if !allowed[t] {
			return fmt.Errorf("The %s '%s' is not recognized", key, t)
		}
	}

	if len(items) == 1 {
		s.Query[key] = items[0]
	} else {
		s.Query[key] = map[string][]string{"$in": items}
	}

	return nil
}

func (s *Search) ParseMultiverseId() error {
	mid := s.Args.Get("multiverseid")

	if mid == "" {
		return nil
	}

	id, err := strconv.Atoi(mid)

	if err == nil {
		s.Query["editions.multiverseid"] = id
	}
	return err
}

func (s *Search) ParseSupertypes() error {
	return s.extractStrings("supertype", "supertypes", map[string]bool{
		"legendary": true,
		"basic":     true,
		"world":     true,
		"snow":      true,
		"ongoing":   true,
	})
}

func (s *Search) ParseSubtypes() error {
	sts := s.Args["subtype"]

	if len(sts) > 0 {
		s.Query["subtypes"] = map[string][]string{"$in": sts}
	}
	return nil
}

func (s *Search) ParseColors() error {
	return s.extractStrings("color", "colors", map[string]bool{
		"red":   true,
		"black": true,
		"blue":  true,
		"white": true,
		"green": true,
	})
}

func (s *Search) ParseRarity() error {
	return s.extractStrings("rarity", "editions.rarity", map[string]bool{
		"common":      true,
		"uncommon":    true,
		"rare":        true,
		"mythic rare": true,
		"special":     true,
		"basic land":  true,
	})
}

func (s *Search) ParseTypes() error {
	err := s.extractStrings("type", "types", map[string]bool{
		"creature":     true,
		"land":         true,
		"tribal":       true,
		"phenomenon":   true,
		"summon":       true,
		"enchantment":  true,
		"sorcery":      true,
		"vanguard":     true,
		"instant":      true,
		"planeswalker": true,
		"artifact":     true,
		"plane":        true,
		"scheme":       true,
	})

	if err != nil {
		return err
	}

	if _, set := s.Query["types"]; !set {
		s.Query["types"] = map[string][]string{
			"$in": []string{"creature", "land", "enchantment",
				"sorcery", "instant", "planeswalker", "artifact"},
		}
	}

	return nil
}

func ParseSearch(u *url.URL) (interface{}, error, []string) {
	search := Search{Args: u.Query(), Query: map[string]interface{}{}}

	errs := []error{
		search.ParseMultiverseId(),
		search.ParseRarity(),
		search.ParseTypes(),
		search.ParseSupertypes(),
		search.ParseColors(),
		search.ParseSubtypes(),
	}

	var err error
	results := []string{}

	for _, e := range errs {
		if e != nil {
			results = append(results, e.Error())
			err = fmt.Errorf("Errors while processing the search")
		}
	}

	return search.Query, err, results
}

func CardsPaging(u *url.URL) (int, error) {
	pagenum := u.Query().Get("page")

	if pagenum == "" {
		pagenum = "0"
	}

	page, err := strconv.Atoi(pagenum)

	if err != nil {
		return 0, err
	}

	if page < 0 {
		return 0, fmt.Errorf("Page parameter must be >= 0")
	}

	return page, nil
}
