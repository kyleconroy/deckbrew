package brew

import (
	"fmt"
	"strings"

	"github.com/kyleconroy/deckbrew/config"
)

type router struct {
	cfg *config.Config
}

func base(host string) string {
	if strings.Contains(host, ":") {
		return "http://" + host
	} else {
		return "https://" + host
	}
}

func (r router) CardURL(id string) string {
	return fmt.Sprintf("%s/mtg/cards/%s", base(r.cfg.HostAPI), id)
}

func (r router) EditionURL(id int) string {
	return fmt.Sprintf("%s/mtg/cards?multiverseid=%d", base(r.cfg.HostAPI), id)
}

func (r router) SetURL(id string) string {
	return fmt.Sprintf("%s/mtg/sets/%s", base(r.cfg.HostAPI), id)
}

func (r router) SetCardsURL(id string) string {
	return fmt.Sprintf("%s/mtg/cards?set=%s", base(r.cfg.HostAPI), id)
}

func (r router) EditionImageURL(id int) string {
	return fmt.Sprintf("%s/mtg/multiverseid/%d.jpg", base(r.cfg.HostImage), id)
}

func (r router) EditionHtmlURL(id int) string {
	return fmt.Sprintf("%s/mtg/cards/%d", base(r.cfg.HostWeb), id)
}
