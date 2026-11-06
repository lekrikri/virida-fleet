package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "") // "" = lire PGHOST, PGUSER, PGPASSWORD, PGDATABASE
	if err != nil {
		fmt.Fprintln(os.Stderr, "Configuration de la base invalide :", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil { // vérifier que la base répond vraiment
		fmt.Fprintln(os.Stderr, "Base injoignable :", err)
		os.Exit(1)
	}
	fmt.Println("Connecté à la base")

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

}
