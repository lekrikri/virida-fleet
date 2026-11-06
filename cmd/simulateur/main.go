package main

import (
	"encoding/json"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"math"
	"math/rand"
	"os"
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

type Capteur struct {
	ID        string
	Nom       string
	Unite     string
	Min       float64
	Max       float64
	Valeur    float64
	Variation float64
}

func nouveauxCapteurs() []*Capteur {
	return []*Capteur{
		{Nom: "temperature", Unite: "°C", Min: 18, Max: 30, Variation: 0.3, Valeur: 24},
		{Nom: "humidite", Unite: "%", Min: 40, Max: 60, Variation: 0.5, Valeur: 50},
		{Nom: "ph", Unite: "pH", Min: 6, Max: 7, Variation: 0.05, Valeur: 6.5},
	}
}

func (c *Capteur) Bouger() {
	pas := c.Variation * (rand.Float64()*2 - 1)               // entre -Variation et +Variation
	c.Valeur = math.Max(c.Min, math.Min(c.Max, c.Valeur+pas)) // coincée dans [Min, Max]
}

func main() {

	broker := os.Getenv("MQTT_BROKER")
	if broker == "" {
		broker = "tcp://localhost:1883" // une valeur par défaut, pour tester sur ton PC : "tcp://localhost:1883"
	}

	nomPod, _ := os.Hostname() // dans Kubernetes, c'est le nom du pod
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID("simulateur-" + nomPod).
		SetConnectRetry(true) // réessayer si le broker n'est pas encore prêt
	client := mqtt.NewClient(opts)
	jeton := client.Connect()
	jeton.Wait()
	if err := jeton.Error(); err != nil {
		fmt.Fprintln(os.Stderr, "Erreur de connexion MQTT:", err)
		os.Exit(1)
	}

	serres := map[string][]*Capteur{
		"serre-1": nouveauxCapteurs(),
		"serre-2": nouveauxCapteurs(),
		"serre-3": nouveauxCapteurs(),
	}
	compteur := 0
	ticker := time.NewTicker(2 * time.Second)
	for range ticker.C {
		for nom, capteurs := range serres {
			for _, c := range capteurs {
				c.Bouger() // faire bouger le capteur
				compteur++
				m := Mesure{
					ID:        fmt.Sprintf("mesure-%d", compteur),
					Serre:     nom,                            // le nom de la serre
					Capteur:   c.Nom,                          // le nom du capteur
					Valeur:    math.Round(c.Valeur*100) / 100, // arrondi à 2 décimales
					Unite:     c.Unite,
					Timestamp: time.Now(),
				}
				jsonData, err := json.Marshal(m)
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error:", err)
					continue
				}
				fmt.Fprintln(os.Stdout, string(jsonData))
				sujet := fmt.Sprintf("virida/%s/%s", nom, c.Nom) // la serre, puis le capteur
				client.Publish(sujet, 0, false, jsonData)
			}
		}
	}
}
