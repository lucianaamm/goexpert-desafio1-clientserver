package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	serverURL     = "http://localhost:8080/cotacao"
	clientTimeout = 300 * time.Millisecond
	outputFile    = "cotacao.txt"
)

type quoteResponse struct {
	Bid string `json:"bid"`
}

func main() {
	bid, err := fetchBid()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("timeout ao consultar servidor: %v", err)
		}
		os.Exit(1)
	}

	if err := saveToFile(bid); err != nil {
		log.Fatalf("erro ao salvar arquivo: %v", err)
	}
}

func fetchBid() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), clientTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("servidor retornou status %d", resp.StatusCode)
	}

	var quote quoteResponse
	if err := json.Unmarshal(body, &quote); err != nil {
		return "", err
	}

	return quote.Bid, nil
}

func saveToFile(bid string) error {
	content := fmt.Sprintf("Dólar: %s", bid)
	return os.WriteFile(outputFile, []byte(content), 0o644)
}
