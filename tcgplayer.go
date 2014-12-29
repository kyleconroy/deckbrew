package main

import (
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"code.google.com/p/go.net/html"
)

func ReplaceUnicode(name string) string {
	replace := map[string]string{
		"Æ": "AE",
		"é": "e",
		"ö": "o",
		"û": "u",
		"á": "a",
		"â": "a",
		"ú": "u",
		"à": "a",
	}
	s := name

	for unicode, ascii := range replace {
		s = strings.Replace(s, unicode, ascii, -1)
	}

	return s
}

func TCGSlug(name string) string {
	re := regexp.MustCompile(`[,.'"?:]`)
	d := strings.Replace(strings.ToLower(name), " ", "-", -1)
	return ReplaceUnicode(re.ReplaceAllLiteralString(d, ""))
}

func TCGCardURL(c *Card) string {
	if len(c.Editions) == 1 {
		return TCGEditionURL(c, &c.Editions[0])
	} else {
		return "http://store.tcgplayer.com/magic/product/show?partner=DECKBREW&ProductName=" + TCGSlug(c.Name)
	}
}

func TCGEditionURL(c *Card, e *Edition) string {
	set := TCGSlug(TCGSet(e.SetId, e.Set))
	id := TCGSlug(TCGName(c.Name, e.MultiverseId))
	return fmt.Sprintf("http://store.tcgplayer.com/magic/%s/%s?partner=DECKBREW", set, id)
}

type TCGPrice struct {
	High    int
	Low     int
	Average int
	Name    string
}

func (t *TCGPrice) Convert() Price {
	return Price{
		High:    int(t.High * 100),
		Low:     int(t.Low * 100),
		Average: int(t.Average * 100),
	}
}

type PriceList struct {
	Prices map[string]Price
}

func (pl *PriceList) GetPrice(mid int) *Price {
	p, ok := pl.Prices[strconv.Itoa(mid)]

	if ok {
		return &p
	} else {
		return nil
	}
}

func NormalizeName(name string) string {
	return strings.ToLower(ReplaceUnicode(strings.TrimSpace(name)))
}

func TCGName(name string, id int) string {
	translation := TranslateID(id)
	if translation != "" {
		return NormalizeName(translation)
	} else {
		return NormalizeName(name)
	}
}

func loadPriceGuide(setName string) (map[string]Price, error) {
	u := "http://magic.tcgplayer.com/db/price_guide.asp?setname=" + url.QueryEscape(setName)
	resp, err := http.Get(u)
	if err != nil {
		return map[string]Price{}, err
	}
	return ParsePriceGuide(resp.Body)
}

func ScrapePrices(db *sql.DB, set Set) (map[string]Price, error) {
	finalPrices := map[string]Price{}

	if !set.Priced {
		return finalPrices, fmt.Errorf("set %s isn't priced", set.Name)
	}

	prices, err := loadPriceGuide(set.PriceGuide)
	if err != nil {
		return prices, err
	}

	// TCGPLayer is really stupid, and has two lists for each duel deck. To fix this,
	// we try to load both deck names.
	if strings.Contains(set.PriceGuide, " vs. ") {
		more, err := loadPriceGuide(strings.Replace(set.PriceGuide, " vs. ", " vs ", 1))
		if err != nil {
			return more, err
		}
		for k, v := range more {
			prices[k] = v
		}
	}

	return associatePrices(db, set, prices)
}

func associatePrices(db *sql.DB, set Set, prices map[string]Price) (map[string]Price, error) {
	finalPrices := map[string]Price{}

	rows, err := db.Query("SELECT record FROM cards WHERE sets @> $1", strings.ToLower("{"+set.Id+"}"))

	if err != nil {
		return finalPrices, err
	}

	cards, err := scanCards(rows)

	if err != nil {
		return finalPrices, err
	}

	if len(cards) == 0 {
		return finalPrices, fmt.Errorf("No cards in set")
	}

	for _, c := range cards {

		// Skip basic lands
		if len(c.Supertypes) == 1 && c.Supertypes[0] == "basic" {
			continue
		}

		var e Edition
		found := false

		for _, edition := range c.Editions {
			if edition.SetId == set.Id {
				e = edition
				found = true
			}
		}

		if !found {
			log.Println("Can't find edition for set")
			continue
		}

		if e.Layout == "plane" && e.SetId == "PC2" {
			continue
		}
		// TCGPlayer doesn't support back side
		if strings.HasSuffix(e.Number, "b") && e.Layout == "double-faced" {
			continue
		}

		// TCGPlayer doesn't support bottom side lookup")
		if strings.HasSuffix(e.Number, "b") && e.Layout == "flip" {
			continue
		}

		tcgname := TCGName(c.Name, e.MultiverseId)

		if _, ok := prices[tcgname]; !ok {
			if e.Layout != "vanguard" {
				log.Println("NOT FOUND", e.SetId, set.Name, strconv.QuoteToASCII(c.Name))
			}
			continue
		}

		finalPrices[strconv.Itoa(e.MultiverseId)] = prices[tcgname]
	}

	return finalPrices, nil
}

func parseMoney(dollar string) int {
	for _, symbol := range []string{".", "$", ","} {
		dollar = strings.Replace(dollar, symbol, "", -1)
	}
	money, err := strconv.Atoi(strings.TrimSpace(dollar))
	if err != nil {
		return 0
	}
	return money
}

func ParsePriceGuide(page io.Reader) (map[string]Price, error) {
	doc, err := html.Parse(page)

	results := map[string]Price{}

	if err != nil {
		return results, err
	}

	tables := FindAll(doc, "table")

	if len(tables) < 3 {
		return results, fmt.Errorf("Couldn't find the third pricing table")
	}

	for _, row := range FindAll(tables[2], "tr") {
		tds := FindAll(row, "td")

		if len(tds) != 8 {
			return results, fmt.Errorf("A proper pricing table has 8 cells")
		}

		name := NormalizeName(Flatten(tds[0]))

		if strings.HasPrefix(name, "dand") {
			log.Println(strconv.QuoteToASCII(name))
		}

		h := parseMoney(Flatten(tds[5]))
		a := parseMoney(Flatten(tds[6]))
		l := parseMoney(Flatten(tds[7]))

		// Handle split cards
		if strings.Contains(name, "//") {
			for _, part := range strings.Split(name, "//") {
				results[strings.TrimSpace(part)] = Price{High: h, Average: a, Low: l}
			}
		} else if strings.Contains(name, "/") {
			for _, part := range strings.Split(name, "/") {
				results[strings.TrimSpace(part)] = Price{High: h, Average: a, Low: l}
			}
		} else {
			results[name] = Price{High: h, Average: a, Low: l}
		}
	}

	return results, nil
}

type scrapeResult struct {
	ID    string
	Price Price
}

func fetchPrices(db *sql.DB, sets []Set) map[string]Price {
	var wg sync.WaitGroup

	pipeline := make(chan scrapeResult, 100)

	for _, set := range sets {
		if !set.Priced {
			continue
		}
		wg.Add(1)
		go func(s Set) {
			defer wg.Done()
			p, e := ScrapePrices(db, s)

			if e != nil {
				log.Println(e)
				return
			}

			for _, price := range p {
				pipeline <- scrapeResult{ID: s.Id, Price: price}
			}
		}(set)
	}

	go func() {
		wg.Wait()
		close(pipeline)
	}()

	prices := map[string]Price{}
	for result := range pipeline {
		prices[result.ID] = result.Price
	}
	return prices
}

func loadPrices(db *sql.DB) (map[string]Price, error) {
	sets, err := FetchSets(db)
	if err != nil {
		return map[string]Price{}, err
	}
	return fetchPrices(db, sets), nil
}

func insertPrices(db *sql.DB, older, newer map[string]Price) error {
	for id, newPrice := range newer {
		// Skip if the price hasn't changed
		if true &&
			older[id].Low == newPrice.Low &&
			older[id].Average == newPrice.Average &&
			older[id].High == newPrice.High {
			continue
		}

		err := InsertPrice(db, id, newPrice)
		if err != nil {
			return err
		}
	}
	return nil
}

func SyncPrices() error {
	db, err := getDatabase()
	if err != nil {
		return err
	}

	savedPrices, err := FetchPrices(db)
	if err != nil {
		return err
	}

	prices, err := loadPrices(db)
	if err != nil {
		return err
	}

	return insertPrices(db, savedPrices, prices)
}

func savePriceGuide(id, name string) error {
	guide := filepath.Join("prices", strings.Replace(name, "/", "", -1)+".html")
	if _, err := os.Stat(guide); err == nil {
		return nil
	}
	u := "http://magic.tcgplayer.com/db/price_guide.asp?setname=" + url.QueryEscape(name)
	log.Printf("saving prices set=%s\n", id)
	resp, err := http.Get(u)

	if err != nil {
		return err
	}

	blob, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(guide, blob, 0777)
}

func ValidatePrices() error {
	db, err := getDatabase()
	if err != nil {
		return err
	}
	sets, err := FetchSets(db)
	if err != nil {
		return err
	}
	for _, set := range sets {
		if !set.Priced {
			continue
		}

		if err := savePriceGuide(set.Id, set.PriceGuide); err != nil {
			return err
		}

		guide := filepath.Join("prices", strings.Replace(set.PriceGuide, "/", "", -1)+".html")
		file, err := os.Open(guide)
		if err != nil {
			return err
		}

		prices, err := ParsePriceGuide(file)
		if err != nil {
			return err
		}

		// TCGPLayer is really stupid, and has two lists for each duel deck. To fix this,
		// we try to load both deck names.
		if strings.Contains(set.PriceGuide, " vs. ") {
			name := strings.Replace(set.PriceGuide, " vs. ", " vs ", 1)

			if err := savePriceGuide(set.Id, name); err != nil {
				return err
			}

			guide := filepath.Join("prices", strings.Replace(name, "/", "", -1)+".html")
			file, err := os.Open(guide)
			if err != nil {
				return err
			}

			more, err := ParsePriceGuide(file)
			if err != nil {
				return err
			}
			for k, v := range more {
				prices[k] = v
			}
		}

		_, err = associatePrices(db, set, prices)
		if err != nil {
			return err
		}
	}
	return nil
}
