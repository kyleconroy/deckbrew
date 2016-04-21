package api

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/kyleconroy/deckbrew/brew"
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
	Conditions []brew.Condition
	Args       url.Values
	Search     brew.Search
}

func (s *Search) extractPattern(searchTerm, key string) error {
	or := []brew.Condition{}

	for _, oracle := range s.Args[key] {
		if oracle == "" {
			continue
		}
		if strings.ContainsAny(oracle, "%_") {
			return fmt.Errorf("Search string can't contain '%%' or '_'")
		}
		or = append(or, brew.ILike(searchTerm, "%"+oracle+"%"))
	}

	if len(or) > 0 {
		s.Conditions = append(s.Conditions, brew.Or(or...))
	}

	return nil
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

func ParseMulticolor(s *brew.Search, args url.Values) error {
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

func ParseMultiverseIDs(s *brew.Search, args url.Values) error {
	ids := args["multiverseid"][:]
	for _, m := range args["m"] {
		ids = append(ids, m)
	}
	s.MultiverseIDs = ids
	return nil
}

func ParseSupertypes(s *brew.Search, args url.Values) (err error) {
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
func ParseSubtypes(s *brew.Search, args url.Values) error {
	s.Subtypes = toLower(args["subtype"])
	return nil
}

// TODO: Add validation
func ParseSets(s *brew.Search, args url.Values) error {
	s.Sets = toLower(args["set"])
	return nil
}

func ParseColors(s *brew.Search, args url.Values) (err error) {
	s.Colors, err = extractStrings(args, "color", map[string]bool{
		"red":   true,
		"black": true,
		"blue":  true,
		"white": true,
		"green": true,
	})
	return
}

func ParseStatus(s *brew.Search, args url.Values) (err error) {
	s.Status, err = extractStrings(args, "status", map[string]bool{
		"legal":      true,
		"banned":     true,
		"restricted": true,
	})
	return
}

func ParseFormat(s *brew.Search, args url.Values) (err error) {
	s.Formats, err = extractStrings(args, "format", map[string]bool{
		"commander": true,
		"standard":  true,
		"modern":    true,
		"vintage":   true,
		"legacy":    true,
	})
	return
}
func ParseRarity(s *brew.Search, args url.Values) (err error) {
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

func ParseTypes(s *brew.Search, args url.Values) (err error) {
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

func ParseName(s *brew.Search, args url.Values) (err error) {
	s.Names, err = extractPattern(args, "name")
	return
}

func ParseRules(s *brew.Search, args url.Values) (err error) {
	s.Rules, err = extractPattern(args, "oracle")
	return
}

func ParseSearch(u *url.URL) (brew.Search, error, []string) {
	args := u.Query()
	search := brew.Search{}

	funcs := []func(*brew.Search, url.Values) error{
		ParseMulticolor,
		ParseRarity,
		ParseTypes,
		ParseSupertypes,
		ParseColors,
		ParseSubtypes,
		ParseFormat,
		ParseStatus,
		ParseMultiverseIDs,
		ParseSets,
		ParseName,
		ParseRules,
	}

	var err error
	results := []string{}

	for _, fun := range funcs {
		if e := fun(&search, args); e != nil {
			results = append(results, e.Error())
			err = fmt.Errorf("Errors while processing the search")
		}
	}

	return search, err, results
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
