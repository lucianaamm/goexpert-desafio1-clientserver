package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	apiBaseURL = "https://economia.awesomeapi.com.br/json/last/"
	apiTimeout = 200 * time.Millisecond
	dbTimeout  = 10 * time.Millisecond
	serverPort = ":8080"
	dbPath     = "./data/cotacao.db"
)

var pairFormat = regexp.MustCompile(`^[A-Z0-9]+-[A-Z0-9]+$`)

type Quote struct {
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
		handleCotacao(w, r, db)
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
			par TEXT NOT NULL,
			bid TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	_, _ = db.Exec(`ALTER TABLE cotacoes ADD COLUMN par TEXT NOT NULL DEFAULT 'USD-BRL'`)
	return nil
}

func validatePair(pair string) error {
	if pair == "" {
		return errors.New("informe o par de moedas via query param moeda. Exemplo: /cotacao?moeda=USD-BRL")
	}
	if !pairFormat.MatchString(pair) {
		return fmt.Errorf("formato inválido: %s (use MOEDA-MOEDA, ex: USD-BRL, BTC-BRL, EUR-USD)", pair)
	}
	return nil
}

func handleCotacao(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	pair := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("moeda")))
	if err := validatePair(pair); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	quote, err := fetchQuote(pair)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("timeout ao consultar API externa: %v", err)
		}
		http.Error(w, "erro ao consultar cotação", http.StatusInternalServerError)
		return
	}

	if err := saveQuote(db, pair, quote.Bid); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("timeout ao persistir no banco: %v", err)
		}
		http.Error(w, "erro ao persistir cotação", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(quote); err != nil {
		log.Printf("erro ao enviar resposta: %v", err)
	}
}

func fetchQuote(pair string) (*Quote, error) {
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBaseURL+pair, nil)
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

	var data map[string]Quote
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	key := strings.ReplaceAll(pair, "-", "")
	quote, ok := data[key]
	if !ok {
		return nil, fmt.Errorf("cotação não encontrada para %s", pair)
	}

	return &quote, nil
}

func saveQuote(db *sql.DB, pair, bid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	_, err := db.ExecContext(ctx, "INSERT INTO cotacoes (par, bid) VALUES (?, ?)", pair, bid)
	return err
}
