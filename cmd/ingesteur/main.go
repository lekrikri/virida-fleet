package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jackc/pgx/v5/pgxpool"
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

func main() {
	ctx := context.Background()

	// 1. Préparer le pool de connexions (ne se connecte pas encore)
	pool, err := pgxpool.New(ctx, "") // "" = lire PGHOST, PGUSER, PGPASSWORD, PGDATABASE
	if err != nil {
		fmt.Fprintln(os.Stderr, "Configuration de la base invalide :", err)
		os.Exit(1)
	}
	defer pool.Close()

	// 2. Préparer le client MQTT (ne se connecte pas encore)
	broker := os.Getenv("MQTT_BROKER")
	if broker == "" {
		broker = "tcp://localhost:1883"
	}
	nomPod, _ := os.Hostname()
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID("ingesteur-" + nomPod).
		SetConnectRetry(true)
	client := mqtt.NewClient(opts)

	// 3. Sondes de santé : le serveur démarre avant toute attente,
	// pour que Kubernetes reçoive tout de suite « vivant, pas prêt »
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		// État du moment, sans attente : c'est Kubernetes qui réessaie
		ctxPing, annuler := context.WithTimeout(r.Context(), time.Second)
		defer annuler()
		if client.IsConnectionOpen() && pool.Ping(ctxPing) == nil {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}
	go func() {
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			fmt.Fprintln(os.Stderr, "Erreur du serveur HTTP :", err)
			os.Exit(1)
		}
	}()

	// 4. Attendre la base, avec une attente qui double (plafonnée à 15 s).
	// Chaque Ping est limité à 3 s : sans délai, il attendrait indéfiniment.
	var errBase error
	attente := time.Second
	for tentative := 1; tentative <= 10; tentative++ {
		ctxPing, annuler := context.WithTimeout(ctx, 3*time.Second)
		errBase = pool.Ping(ctxPing)
		annuler()
		if errBase == nil {
			break
		}
		fmt.Fprintf(os.Stderr, "Base pas encore prête (essai %d/10), nouvel essai dans %s\n", tentative, attente)
		time.Sleep(attente)
		attente = min(attente*2, 15*time.Second)
	}
	if errBase != nil {
		fmt.Fprintln(os.Stderr, "Base injoignable après 10 essais :", errBase)
		os.Exit(1)
	}
	fmt.Println("Connecté à la base")

	// 5. Créer la table si elle n'existe pas
	const schema = `
CREATE TABLE IF NOT EXISTS mesures (
	id          UUID             NOT NULL,
	site        TEXT             NOT NULL,
	serre       TEXT             NOT NULL,
	capteur     TEXT             NOT NULL,
	valeur      DOUBLE PRECISION NOT NULL,
	unite       TEXT             NOT NULL,
	horodatage  TIMESTAMPTZ      NOT NULL,
	PRIMARY KEY (id, horodatage)
);
SELECT create_hypertable('mesures', by_range('horodatage'), if_not_exists => TRUE);`
	if _, err := pool.Exec(ctx, schema); err != nil {
		fmt.Fprintln(os.Stderr, "Erreur lors de la création de la table :", err)
		os.Exit(1)
	}

	// 6. Se connecter à MQTT et s'abonner
	jeton := client.Connect()
	jeton.Wait()
	if err := jeton.Error(); err != nil {
		fmt.Fprintln(os.Stderr, "Erreur de connexion MQTT :", err)
		os.Exit(1)
	}
	jeton = client.Subscribe("virida/#", 1, func(c mqtt.Client, msg mqtt.Message) {
		var m Mesure
		if err := json.Unmarshal(msg.Payload(), &m); err != nil {
			fmt.Fprintln(os.Stderr, "Message illisible :", err)
			return
		}
		_, err := pool.Exec(ctx,
			`INSERT INTO mesures (id, site, serre, capteur, valeur, unite, horodatage)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 ON CONFLICT (id, horodatage) DO NOTHING`,
			m.ID, m.Site, m.Serre, m.Capteur, m.Valeur, m.Unite, m.Timestamp)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Insertion impossible :", err)
		}
	})
	jeton.Wait()

	// 7. Attendre les mesures
	fmt.Println("Abonné à virida/#, en attente de mesures")
	select {}
}
