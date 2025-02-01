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

	"github.com/joho/godotenv"
)

type Coordinate struct {
	Place string  `json:"-"`
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
}

func (c Coordinate) String() string {
	return fmt.Sprintf("(%f, %f)", c.Lat, c.Lon)
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
	apiKeys, err := godotenv.Read()
	if err != nil {
		log.Fatalln("Error: can't load environment or .env file")
	}

	apiKey, exist := apiKeys["OPENWEATHERMAP_API_KEY"]
	if !exist {
		log.Fatalln("Error: OPENWEATHERMAP_API_KEY does not existed!")
	}
	if strings.TrimSpace(apiKey) == "" {
		log.Fatalln("Error: OPENWEATHERMAP_API_KEY can't be empty!")
	}
	return apiKey
}

// Fetch weather data with specific latitude and longitude and decode into struct
func fetchWeatherData(lat, lon float32, apiKey string) DataEntry {
	var url = fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?lat=%f&lon=%f&appid=%s", lat, lon, strings.TrimSpace(apiKey))
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

// https://www.stackhawk.com/blog/golang-cors-guide-what-it-is-and-how-to-enable-it/
func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

// lat, lon, rain+place
// state-id, rainfall (scale 100)

func rawDataHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	apiKey := loadEnv()

	coordCh := make(chan Coordinate)
	go readCoordData("data/test.csv", coordCh)

	var max float32 = 0.0
	var min float32 = 10000000.0

	item := strings.TrimSpace(r.URL.Query().Get("scale"))
	scale, _ := strconv.ParseBool(item)

	var entries []DataEntry
	for coord := range coordCh {
		entry := fetchWeatherData(coord.Lat, coord.Lon, apiKey)
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
	http.HandleFunc("/api/data", rawDataHandler) // /api/data?scale=100
	log.Fatal(http.ListenAndServe(":8080", nil))
}
