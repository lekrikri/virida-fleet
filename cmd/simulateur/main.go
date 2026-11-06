package main

import (
	"encoding/json"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type Mesure struct {
	ID        string    `json:"id"`
	Site      string    `json:"site"`
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

	site := os.Getenv("SITE")
	if site == "" {
		site = "local"
	}

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
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if client.IsConnectionOpen() { // la connexion MQTT est-elle ouverte ?
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})
	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			fmt.Fprintln(os.Stderr, "Erreur du serveur HTTP:", err)
			os.Exit(1)
		}
	}()

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

	ticker := time.NewTicker(2 * time.Second)
	for range ticker.C {
		for nom, capteurs := range serres {
			for _, c := range capteurs {
				c.Bouger() // faire bouger le capteur
				m := Mesure{
					ID:        uuid.New().String(),
					Site:      site,
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
				sujet := fmt.Sprintf("virida/%s/%s/%s", site, nom, c.Nom)

				client.Publish(sujet, 0, false, jsonData)
			}
		}
	}
}
