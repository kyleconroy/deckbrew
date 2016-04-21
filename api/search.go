package api

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/kyleconroy/deckbrew/brew"
)

func toLower(strs []string) []string {
	downers := []string{}
	for _, s := range strs {
		downers = append(downers, strings.ToLower(s))
	}
	return downers
}

func extractPattern(args url.Values, key string) ([]string, error) {
	items := []string{}
	for _, oracle := range args[key] {
		if oracle == "" {
			continue
		}
		if strings.ContainsAny(oracle, "%_") {
			return items, fmt.Errorf("Search string can't contain '%%' or '_'")
		}
		items = append(items, "%"+oracle+"%")
	}
	return items, nil
}

func extractStrings(args url.Values, key string, allowed map[string]bool) ([]string, error) {
	items := args[key]

	for _, t := range items {
		if !allowed[t] {
			return items, fmt.Errorf("The %s '%s' is not recognized", key, t)
		}
	}

	return items, nil
}

func parseMulticolor(s *brew.Search, args url.Values) error {
	switch args.Get("multicolor") {
	case "true":
		s.IncludeMulticolor = true
		s.Multicolor = true
	case "false":
		s.IncludeMulticolor = true
		s.Multicolor = false
	case "":
		s.IncludeMulticolor = false
		return nil
	default:
		return fmt.Errorf("Multicolor should be either 'true' or 'false'")
	}
	return nil
}

func parseMultiverseIDs(s *brew.Search, args url.Values) error {
	ids := args["multiverseid"][:]
	for _, m := range args["m"] {
		ids = append(ids, m)
	}
	s.MultiverseIDs = ids
	return nil
}

func parseSupertypes(s *brew.Search, args url.Values) (err error) {
	s.Supertypes, err = extractStrings(args, "supertype", map[string]bool{
		"legendary": true,
		"basic":     true,
		"world":     true,
		"snow":      true,
		"ongoing":   true,
	})
	return
}

// TODO: Add validation
func parseSubtypes(s *brew.Search, args url.Values) error {
	s.Subtypes = toLower(args["subtype"])
	return nil
}

// TODO: Add validation
func parseSets(s *brew.Search, args url.Values) error {
	s.Sets = toLower(args["set"])
	return nil
}

func parseColors(s *brew.Search, args url.Values) (err error) {
	s.Colors, err = extractStrings(args, "color", map[string]bool{
		"red":   true,
		"black": true,
		"blue":  true,
		"white": true,
		"green": true,
	})
	return
}

func parseStatus(s *brew.Search, args url.Values) (err error) {
	s.Status, err = extractStrings(args, "status", map[string]bool{
		"legal":      true,
		"banned":     true,
		"restricted": true,
	})
	return
}

func parseFormat(s *brew.Search, args url.Values) (err error) {
	s.Formats, err = extractStrings(args, "format", map[string]bool{
		"commander": true,
		"standard":  true,
		"modern":    true,
		"vintage":   true,
		"legacy":    true,
	})
	return
}

func parseRarity(s *brew.Search, args url.Values) (err error) {
	s.Rarities, err = extractStrings(args, "rarity", map[string]bool{
		"common":   true,
		"uncommon": true,
		"rare":     true,
		"mythic":   true,
		"special":  true,
		"basic":    true,
	})
	return
}

func parseTypes(s *brew.Search, args url.Values) (err error) {
	s.Types, err = extractStrings(args, "type", map[string]bool{
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
	return
}

func parseName(s *brew.Search, args url.Values) (err error) {
	s.Names, err = extractPattern(args, "name")
	return
}

func parseRules(s *brew.Search, args url.Values) (err error) {
	s.Rules, err = extractPattern(args, "oracle")
	return
}

func parsePaging(s *brew.Search, args url.Values) error {
	s.Limit = 100

	pagenum := args.Get("page")
	if pagenum == "" {
		return nil
	}

	page, err := strconv.Atoi(pagenum)
	if err != nil {
		return err
	}

	if page < 0 {
		return fmt.Errorf("Page parameter must be >= 0")
	}

	s.Page = page
	s.Offset = s.Page * s.Limit

	return nil
}

func ParseSearch(u *url.URL) (brew.Search, error, []string) {
	args := u.Query()
	search := brew.Search{}

	funcs := []func(*brew.Search, url.Values) error{
		parseMulticolor,
		parseRarity,
		parseTypes,
		parseSupertypes,
		parseColors,
		parseSubtypes,
		parseFormat,
		parseStatus,
		parseMultiverseIDs,
		parseSets,
		parseName,
		parseRules,
		parsePaging,
	}

	var err error
	results := []string{}

	for _, fun := range funcs {
		if e := fun(&search, args); e != nil {
			results = append(results, e.Error())
			err = fmt.Errorf("Errors while processing the search")
		}
	}

	// By default, include 100 cards
	search.Limit = 100

	return search, err, results
}
