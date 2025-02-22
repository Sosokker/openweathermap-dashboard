package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/rs/cors"
)

var c *cache.Cache

func init() {
	c = cache.New(30*time.Minute, 60*time.Minute)
	// c.Set("coordinate", "entry", cache.DefaultExpiration)
}

type Coordinate struct {
	Place string  `json:"place"`
	Lon   float32 `json:"lon"`
	Lat   float32 `json:"lat"`
}

type Weather struct {
	Id          int    `json:"id"`
	Main        string `json:"main"`
	Description string `json:"description"`
}

type DataEntry struct {
	Name    string     `json:"name"`
	Coord   Coordinate `json:"coord"`
	Weather []Weather  `json:"weather"`
	Rain    struct {
		PerHour float32 `json:"1h"`
	} `json:"rain"`
	State string `json:"state,omitempty"`
}

func (c Coordinate) String() string {
	return fmt.Sprintf("(%f, %f, %s)", c.Lat, c.Lon, c.Place)
}

func (w Weather) String() string {
	return fmt.Sprintf("(%d, %s, %s)", w.Id, w.Main, w.Description)
}

func (d DataEntry) String() string {
	return fmt.Sprintf("Place Name: %s\nCoordinate: %s\nWeather: %s\nRain per Hour: %f\n", d.Name, d.Coord, d.Weather, d.Rain.PerHour)
}

func (d *DataEntry) normalizeRainPerHour(min, max float32) {
	if max == 0 {
		return
	}
	d.Rain.PerHour = (float32(d.Rain.PerHour) - min) / (max - min) * 100
}

func loadEnv() string {
	apiKey := os.Getenv("OPENWEATHERMAP_API_KEY")

	if strings.TrimSpace(apiKey) == "" {
		log.Fatalln("Error: OPENWEATHERMAP_API_KEY can't be empty!")
	}
	return apiKey
}

// Fetch weather data with specific latitude and longitude and decode into struct
func fetchWeatherData(lat, lon float32, apiKey string) DataEntry {
	var url = fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?lat=%f&lon=%f&appid=%s", lat, lon, strings.TrimSpace(apiKey))

	coordstr := fmt.Sprintf("%f, %f", lat, lon)
	if data, found := c.Get(coordstr); found {
		return data.(DataEntry)
	}

	fmt.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln("Error:", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalln("Error: Status not okay: ", resp.StatusCode)
	}

	var entry DataEntry
	err = json.NewDecoder(resp.Body).Decode(&entry)
	if err != nil {
		log.Fatalln("Error: error decode json to struct", err)
	}

	// cache
	c.Set(coordstr, entry, cache.DefaultExpiration)

	return entry
}

func readCoordData(filepath string, coordCh chan<- Coordinate) {
	file, err := os.Open(filepath)
	if err != nil {
		log.Fatalln("Error: unable to read file", err)
	}
	defer file.Close()

	r := csv.NewReader(file)
	for {
		line, err := r.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalln("Error: error while reading file", err)
		}

		place, lat, lon := line[0], line[1], line[2]
		place = strings.TrimSpace(place)
		latf, err := strconv.ParseFloat(strings.TrimSpace(lat), 32)
		if err != nil {
			// log.Fatalln("Error: unable to convert string to float", err)
			continue
		}

		lonf, err := strconv.ParseFloat(strings.TrimSpace(lon), 32)
		if err != nil {
			// log.Fatalln("Error: unable to convert string to float", err)
			continue
		}

		coordCh <- Coordinate{place, float32(lonf), float32(latf)}
	}
	close(coordCh)
}

// lat, lon, rain+place
// state-id, rainfall (scale 100)

func rawDataHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := loadEnv()

	coordCh := make(chan Coordinate)
	go readCoordData("data/data.csv", coordCh)

	var max float32 = 0.0
	var min float32 = 10000000.0

	item := strings.TrimSpace(r.URL.Query().Get("scale"))
	scale, _ := strconv.ParseBool(item)

	var entries []DataEntry
	for coord := range coordCh {

		entry := fetchWeatherData(coord.Lat, coord.Lon, apiKey)
		entry.Coord.Place = coord.Place // fetched data not have place field
		if entry.Rain.PerHour > max {
			max = entry.Rain.PerHour
		}
		if entry.Rain.PerHour < min {
			min = entry.Rain.PerHour
		}

		if scale {
			entry.normalizeRainPerHour(min, max)
		}
		entries = append(entries, entry)
	}

	if err := json.NewEncoder(w).Encode(&entries); err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		return
	}
}

func main() {
	// https://www.stackhawk.com/blog/golang-cors-guide-what-it-is-and-how-to-enable-it/
	mux := http.NewServeMux()
	mux.HandleFunc("/api/data", rawDataHandler) // /api/data?scale=100

	handler := cors.Default().Handler(mux)
	// c := cors.New(cors.Options{
	// 	AllowedOrigins:   []string{"*"},
	// 	AllowCredentials: true,
	// 	AllowedHeaders:   []string{"Authorization", "Content-Type", "Access-Control-Allow-Origin"},
	// 	AllowedMethods:   []string{"GET", "UPDATE", "PUT", "POST", "DELETE"},
	// })
	// handler = c.Handler(handler)
	log.Fatal(http.ListenAndServe(":8080", handler))
}
