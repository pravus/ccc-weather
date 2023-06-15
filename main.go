package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"
	"golang.org/x/net/html"
)

type Location struct {
	State  string `json:"state"`
	City   string `json:"city"`
	Field1 string `json:"field1"`
	Field2 string `json:"field2"`
}

type Locations []*Location

const (
	baseURL = `https://forecast.weather.gov/MapClick.php`
)

var (
	gauges = map[string]*prometheus.GaugeVec{
		`F`: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: `weather_temp_f`,
			Help: `Current reported temperature in degrees Fahrenheit`,
		}, []string{`state`, `city`}),
		`C`: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: `weather_temp_c`,
			Help: `Current reported temperature in degrees Celsius`,
		}, []string{`state`, `city`}),
	}

	regexpTemp = regexp.MustCompile(`^(-?\d+)°([CF])$`)
)

func main() {
	var locations Locations
	if err := json.Unmarshal([]byte(os.Getenv(`LOCATIONS`)), &locations); err != nil {
		log.Fatalf(`error unmarshalling locations: %s`, err)
	} else if locations == nil {
		log.Fatalf(`malformed locations configuration`)
	} else if len(locations) <= 0 {
		log.Fatalf(`no locations configured`)
	}

	for _, gauge := range gauges {
		prometheus.MustRegister(gauge)
	}

	http.Handle(`/metrics`, promhttp.Handler())

	c := cron.New()
	for _, location := range locations {
		log.Printf("location: %s %s %s %s\n", location.State, location.City, location.Field1, location.Field2)
		c.AddFunc(`@every 1m`, wrap(location.State, location.City, location.Field1, location.Field2))
	}
	c.Start()

	http.ListenAndServe(`:3300`, nil)
}

func wrap(state, city, field1, field2 string) func() {
	if u, err := url.Parse(baseURL); err != nil {
		log.Fatalf(`probe url: %s`, err)
	} else {
		var q = u.Query()
		q.Add(`textField1`, field1)
		q.Add(`textField2`, field2)
		u.RawQuery = q.Encode()
		return func() { probe(u.String(), state, city) }
	}
	return nil
}

func probe(url, state, city string) {
	var setGauge = func(doc *html.Node, class string) error {
		if value, unit, err := parse(doc, class); err != nil {
			if strings.Contains(err.Error(), `no nodes found`) {
				return nil
			}
			return err
		} else if gauge, found := gauges[unit]; !found {
			return fmt.Errorf(`gauge "%s" not found`, unit)
		} else {
			gauge.WithLabelValues(state, city).Set(value)
			return nil
		}
	}

	if doc, err := htmlquery.LoadURL(url); err != nil {
		log.Printf(`forecast.load: %s, %s %s [%s]`, city, state, err, url)
	} else {
		if err := setGauge(doc, `myforecast-current-lrg`); err != nil {
			log.Printf(`forecast.f: %s, %s %s [%s]`, city, state, err, url)
		}
		if err := setGauge(doc, `myforecast-current-sm`); err != nil {
			log.Printf(`forecast.c: %s, %s %s [%s]`, city, state, err, url)
		}
	}
}

func parse(doc *html.Node, class string) (float64, string, error) {
	if nodes, err := htmlquery.QueryAll(doc, `//*[@class="`+class+`"]`); err != nil {
		return 0, ``, err
	} else if len(nodes) <= 0 {
		return 0, ``, fmt.Errorf(`no nodes found for "%s"`, class)
	} else if text := htmlquery.InnerText(nodes[0]); text == `` || text == `N/A` {
		return 0, ``, fmt.Errorf(`no nodes found for "%s"`, class)
	} else if match := regexpTemp.FindStringSubmatch(text); len(match) <= 2 {
		return 0, ``, fmt.Errorf(`malformed node found for "%s": %s`, class, text)
	} else if v, err := strconv.ParseInt(match[1], 10, 64); err != nil {
		return 0, ``, err
	} else {
		return float64(v), strings.ToUpper(match[2]), nil
	}
}
