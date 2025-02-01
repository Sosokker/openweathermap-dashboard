package main

type Coordinate struct {
	Lon float32 `json:"lon"`
	Lat float32 `json:"lat"`
}

type Weather struct {
	Id          int    `json:"id"`
	Main        string `json:"main"`
	Description string `json:"description"`
}

type DataEntry struct {
	Name    string `json:"name"`
	Coord   Coordinate
	Weather Weather
	Rain    struct {
		PerHour float64 `json:"1h"`
	} `json:"rain"`
}

func main() {

}
