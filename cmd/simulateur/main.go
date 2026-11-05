package main

import (
	"encoding/json"
	"fmt"
	"time"
)

type Mesure struct {
	ID        string    `json:"id"`
	Serre     string    `json:"serre"`
	Capteur   string    `json:"capteur"`
	Valeur    float64   `json:"valeur"`
	Unite     string    `json:"unite"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	mesure := Mesure{
		ID:        "mesure-1",
		Serre:     "serre-1",
		Capteur:   "temperature",
		Valeur:    25.5,
		Unite:     "°C",
		Timestamp: time.Now(),
	}

	jsonData, err := json.Marshal(mesure)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(string(jsonData))
}
