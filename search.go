package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func toUpper(strs []string) []string {
	uppers := []string{}
	for _, s := range strs {
		uppers = append(uppers, strings.ToUpper(s))
	}
	return uppers
}

func toLower(strs []string) []string {
	downers := []string{}
	for _, s := range strs {
		downers = append(downers, strings.ToLower(s))
	}
	return downers
}

type Search struct {
	Conditions []Condition
	Args       url.Values
}

func (s *Search) extractPattern(searchTerm, key string) error {
	or := []Condition{}

	for _, oracle := range s.Args[key] {
		if oracle == "" {
			continue
		}
		if strings.ContainsAny(oracle, "%_") {
			return fmt.Errorf("Search string can't contain '%%' or '_'")
		}
		or = append(or, ILike(searchTerm, "%"+oracle+"%"))
	}

	if len(or) > 0 {
		s.Conditions = append(s.Conditions, Or(or...))
	}

	return nil
}

func (s *Search) extractStrings(searchTerm, key string, allowed map[string]bool) error {
	items := s.Args[searchTerm]

	for _, t := range items {
		if !allowed[t] {
			return fmt.Errorf("The %s '%s' is not recognized", key, t)
		}
	}

	return s.addQuery(key, items)
}

func (s *Search) addQuery(key string, items []string) error {
	if len(items) == 0 {
		return nil
	}

	if len(items) == 1 {
		s.Conditions = append(s.Conditions, Contains(key, CreateStringArray(items)))
	} else {
		s.Conditions = append(s.Conditions, Overlap(key, CreateStringArray(items)))
	}

	return nil
}

func (s *Search) ParseMulticolor() error {
	switch s.Args.Get("multicolor") {
	case "true":
		s.Conditions = append(s.Conditions, Eq("multicolor", "true"))
	case "false":
		s.Conditions = append(s.Conditions, Eq("multicolor", "false"))
	case "":
		return nil
	default:
		return fmt.Errorf("Multicolor should be either 'true' or 'false'")
	}
	return nil
}

func (s *Search) ParseMultiverseId() error {
	return s.addQuery("mids", s.Args["multiverseid"])
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
	return s.addQuery("subtypes", s.Args["subtype"])
}

func (s *Search) ParseSets() error {
	return s.addQuery("sets", toLower(s.Args["set"]))
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

func (s *Search) ParseStatus() error {
	return s.extractStrings("status", "status", map[string]bool{
		"legal":      true,
		"banned":     true,
		"restricted": true,
	})
}

func (s *Search) ParseFormat() error {
	return s.extractStrings("format", "formats", map[string]bool{
		"commander": true,
		"standard":  true,
		"modern":    true,
		"vintage":   true,
		"legacy":    true,
	})
}
func (s *Search) ParseRarity() error {
	return s.extractStrings("rarity", "rarities", map[string]bool{
		"common":   true,
		"uncommon": true,
		"rare":     true,
		"mythic":   true,
		"special":  true,
		"basic":    true,
	})
}

func (s *Search) ParseTypes() error {
	return s.extractStrings("type", "types", map[string]bool{
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
}

func (s *Search) ParseName() error {
	return s.extractPattern("name", "name")
}

func (s *Search) ParseText() error {
	return s.extractPattern("rules", "oracle")
}

func ParseSearch(u *url.URL) (Condition, error, []string) {
	search := Search{Args: u.Query(), Conditions: []Condition{}}

	errs := []error{
		search.ParseMulticolor(),
		search.ParseRarity(),
		search.ParseTypes(),
		search.ParseSupertypes(),
		search.ParseColors(),
		search.ParseSubtypes(),
		search.ParseFormat(),
		search.ParseStatus(),
		search.ParseMultiverseId(),
		search.ParseSets(),
		search.ParseText(),
		search.ParseName(),
	}

	var err error
	results := []string{}

	for _, e := range errs {
		if e != nil {
			results = append(results, e.Error())
			err = fmt.Errorf("Errors while processing the search")
		}
	}

	return And(search.Conditions...), err, results
}

func CardsPaging(u *url.URL) (int, error) {
	pagenum := u.Query().Get("page")

	if pagenum == "" {
		return 0, nil
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
