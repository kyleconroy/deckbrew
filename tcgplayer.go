package main

import (
	"code.google.com/p/go.net/html"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

func ReplaceUnicode(name string) string {
	a := strings.Replace(name, "Æ", "AE", -1)
	e := strings.Replace(a, "é", "e", -1)
	o := strings.Replace(e, "ö", "o", -1)
	return strings.Replace(o, "û", "u", -1)
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
	id := TCGSlug(TCGName(e.MultiverseId, c.Name))
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
	Prices map[int]Price
}

func TCGName(id int, name string) string {
	switch id {
	case 9844:
		return "B.F.M. (Big Furry Monster Right)"
	case 9780:
		return "B.F.M. (Big Furry Monster Left)"
	case 74237:
		return "Our Market Research..."
	case 9757:
		return "The Ultimate Nightmare of Wizards of the Coast Cu"
	default:
		return ReplaceUnicode(name)
	}
}

func TCGSet(setId, set string) string {
	sets := map[string]string{
		"10E": "10th Edition",
		"9ED": "9th Edition",
		"8ED": "8th Edition",
		"7ED": "7th Edition",
		"M14": "Magic 2014 (M14)",
		"M13": "Magic 2013 (M13)",
		"M12": "Magic 2012 (M12)",
		"M11": "Magic 2011 (M11)",
		"M10": "Magic 2010 (M10)",
		"RAV": "Ravnica",
		"DDG": "Duel Decks: Knights vs Dragons",
		"DDL": "Duel Decks: Heroes vs. Monsters",
		"PC2": "Planechase 2012",
		"C13": "Commander 2013",
		"PD2": "Premium Deck Series: Fire and Lightning",
		"LEB": "Beta Edition",
		"LEA": "Alpha Edition",
		"TSB": "Timeshifted",
	}
	if sets[setId] != "" {
		return sets[setId]
	}
	return set
}

func GetPrice(c Card, e Edition) (Price, error) {
	skip := map[string]bool{
		"MED": true,
		"ME2": true,
		"ME3": true,
		"ME4": true,
		"PPR": true,
		"VAN": true,
	}

	if skip[e.SetId] {
		return Price{}, fmt.Errorf("TCGPlayer doesn't support %s", e.Set)
	}

	if e.Rarity == "basic" {
		return Price{}, fmt.Errorf("Basic land pricing isn't needed")
	}

	if e.Layout == "plane" && e.SetId == "PC2" {
		return Price{}, fmt.Errorf("TCGPlayer doesn't list prices for Placechase 2012 planes")
	}

	// FIXME Skipping split cards for now
	if e.Layout == "split" {
		return Price{}, fmt.Errorf("TCGPlayer requires both card names for split cards")
	}

	if strings.HasSuffix(e.Number, "b") && e.Layout == "double-faced" {
		return Price{}, fmt.Errorf("TCGPlayer doesn't support back side lookup")
	}

	if strings.HasSuffix(e.Number, "b") && e.Layout == "flip" {
		return Price{}, fmt.Errorf("TCGPlayer doesn't support bottom side lookup")
	}

	name := TCGName(e.MultiverseId, c.Name)
	v := url.Values{}
	v.Set("pk", "DECKBREW")
	v.Set("s", TCGSet(e.SetId, e.Set))
	v.Set("p", strings.Replace(name, "\"", "", -1))

	resp, err := http.Get("http://partner.tcgplayer.com/x3/phl.asmx/p?" + v.Encode())

	if err != nil {
		return Price{}, err
	}

	var tcg TCGPrice

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return Price{}, err
	}

	err = xml.Unmarshal(body, &tcg)

	if err != nil {
		return Price{}, err
	}

	return tcg.Convert(), nil
}

func UpdatePrices(db *sql.DB, pl *PriceList) {
	rows, err := db.Query("SELECT record FROM cards ORDER BY name")

	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()

	for rows.Next() {
		var blob []byte
		var c Card

		if err := rows.Scan(&blob); err != nil {
			log.Fatal(err)
		}

		err := json.Unmarshal(blob, &c)

		if err != nil {
			log.Fatal(err)
		}

		for _, e := range c.Editions {

			price, err := GetPrice(c, e)

			if err != nil {
				log.Println(c.Name, e.SetId, err)
			}

			pl.Prices[e.MultiverseId] = price
		}

	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}

func LoadPriceList(path string) (PriceList, error) {
	return PriceList{}, nil
}

func ScrapePrices(db *sql.DB, setId, setName string) (map[string]Price, error) {
	log.Println("Starting", setId, setName)
	u := "http://magic.tcgplayer.com/db/price_guide.asp?setname=" + url.QueryEscape(setName)
	resp, err := http.Get(u)

	if err != nil {
		return map[string]Price{}, err
	}

	prices, err := ParsePriceGuide(resp.Body)

	if err != nil {
		return map[string]Price{}, err
	}

	rows, err := db.Query("SELECT record FROM cards WHERE sets @> $1", strings.ToLower("{"+setId+"}"))

	if err != nil {
		return map[string]Price{}, err
	}

	cards, err := scanCards(rows)

	if err != nil {
		return map[string]Price{}, err
	}

	if len(cards) == 0 {
		return map[string]Price{}, fmt.Errorf("No cards in set")
	}
	for _, c := range cards {
        
        // Skip basic lands
		if len(c.Supertypes) == 1 && c.Supertypes[0] == "basic" {
				continue
		}

        e := c.Editions[0]

// TCGPlayer doesn't support back side
	if strings.HasSuffix(e.Number, "b") && e.Layout == "double-faced" {
            continue
	}

// TCGPlayer doesn't support bottom side lookup")
	if strings.HasSuffix(e.Number, "b") && e.Layout == "flip" {
            continue
	}

		if _, ok := prices[strings.ToLower(ReplaceUnicode(c.Name))]; !ok {
			log.Println("NOT FOUND", setName, c.Name)
		}
	}
	return map[string]Price{}, nil

}

func parseMoney(dollar string) int {
	cents := strings.Replace(strings.Replace(dollar, ".", "", -1), "$", "", -1)
	money, err := strconv.Atoi(strings.TrimSpace(cents))
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

		name := strings.ToLower(strings.TrimSpace(Flatten(tds[0])))
		h := parseMoney(Flatten(tds[5]))
		a := parseMoney(Flatten(tds[6]))
		l := parseMoney(Flatten(tds[7]))

		// Handle split cards
		if strings.Contains(name, "//") {
			names := strings.Split(name, "//")
			results[strings.TrimSpace(names[0])] = Price{High: h, Average: a, Low: l}
			results[strings.TrimSpace(names[1])] = Price{High: h, Average: a, Low: l}
		} else {
			results[name] = Price{High: h, Average: a, Low: l}
		}
	}

	return results, nil
}

func DumpPricing(db *sql.DB, output string) error {
	_ = PriceList{Prices: map[int]Price{}}

	scrape := func(set, name string) {
		_, e := ScrapePrices(db, set, name)
		if e != nil {
			log.Fatal(e)
		}
	}

	scrape("ISD", "Innistrad")
	scrape("DKA", "Dark Ascension")
	scrape("AVR", "Avacyn Restored")
	scrape("BNG", "Born of the Gods")
	scrape("THS", "Theros")
	scrape("DGM", "Dragon's Maze")
	scrape("GTC", "Gatecrash")
	scrape("RTR", "Return to Ravnica")

	return nil
}
