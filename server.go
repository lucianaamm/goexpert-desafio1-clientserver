package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

const (
	apiURL     = "https://economia.awesomeapi.com.br/json/last/USD-BRL"
	apiTimeout = 200 * time.Millisecond
	dbTimeout  = 10 * time.Millisecond
	serverPort = ":8080"
	dbPath     = "./data/cotacao.db"
)

type USDQuote struct {
	Code       string `json:"code"`
	Codein     string `json:"codein"`
	Name       string `json:"name"`
	High       string `json:"high"`
	Low        string `json:"low"`
	VarBid     string `json:"varBid"`
	PctChange  string `json:"pctChange"`
	Bid        string `json:"bid"`
	Ask        string `json:"ask"`
	Timestamp  string `json:"timestamp"`
	CreateDate string `json:"create_date"`
}

type apiResponse struct {
	USDBRL USDQuote `json:"USDBRL"`
}

func main() {
	if err := os.MkdirAll("./data", 0o755); err != nil {
		log.Fatalf("erro ao criar diretório data: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("erro ao abrir banco: %v", err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		log.Fatalf("erro ao inicializar banco: %v", err)
	}

	http.HandleFunc("/cotacao", func(w http.ResponseWriter, r *http.Request) {
		handleCotacao(w, db)
	})

	log.Println("servidor rodando na porta 8080")
	if err := http.ListenAndServe(serverPort, nil); err != nil {
		log.Fatalf("erro ao iniciar servidor: %v", err)
	}
}

func initDB(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS cotacoes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			bid TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func handleCotacao(w http.ResponseWriter, db *sql.DB) {
	quote, err := fetchQuote()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("timeout ao consultar API externa: %v", err)
		}
		http.Error(w, "erro ao consultar cotação", http.StatusInternalServerError)
		return
	}

	if err := saveQuote(db, quote.USDBRL.Bid); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("timeout ao persistir no banco: %v", err)
		}
		http.Error(w, "erro ao persistir cotação", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(quote.USDBRL); err != nil {
		log.Printf("erro ao enviar resposta: %v", err)
	}
}

func fetchQuote() (*apiResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var quote apiResponse
	if err := json.Unmarshal(body, &quote); err != nil {
		return nil, err
	}

	return &quote, nil
}

func saveQuote(db *sql.DB, bid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	_, err := db.ExecContext(ctx, "INSERT INTO cotacoes (bid) VALUES (?)", bid)
	return err
}
