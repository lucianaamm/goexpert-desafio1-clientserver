package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	serverBaseURL = "http://localhost:8080/cotacao"
	clientTimeout = 300 * time.Millisecond
	outputFile    = "cotacao.txt"
)

var pairFormat = regexp.MustCompile(`^[A-Z0-9]+-[A-Z0-9]+$`)

type quoteResponse struct {
	Bid  string `json:"bid"`
	Name string `json:"name"`
}

func main() {
	if len(os.Args) < 2 || strings.TrimSpace(os.Args[1]) == "" {
		printUsage()
		os.Exit(1)
	}

	pair := strings.ToUpper(strings.TrimSpace(os.Args[1]))
	if err := validatePair(pair); err != nil {
		log.Print(err)
		printUsage()
		os.Exit(1)
	}

	quote, err := fetchQuote(pair)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("timeout ao consultar servidor: %v", err)
		}
		os.Exit(1)
	}

	label := quote.Name
	if label == "" {
		label = pair
	}

	if err := saveToFile(label, quote.Bid); err != nil {
		log.Fatalf("erro ao salvar arquivo: %v", err)
	}
}

func validatePair(pair string) error {
	if !pairFormat.MatchString(pair) {
		return fmt.Errorf("formato inválido: %s (use MOEDA-MOEDA, ex: USD-BRL, BTC-BRL, EUR-USD)", pair)
	}
	return nil
}

func printUsage() {
	log.Println("informe o par de moedas a ser consultado.")
	log.Println("uso: go run client.go <PAR>")
	log.Println("")
	log.Println("exemplos de pares suportados pela AwesomeAPI:")
	log.Println("  moedas tradicionais: USD-BRL, EUR-BRL, GBP-BRL, JPY-BRL")
	log.Println("  criptomoedas:        BTC-BRL, ETH-BRL, LTC-BRL, XRP-BRL")
	log.Println("  entre estrangeiras:  EUR-USD, GBP-USD, USD-JPY, EUR-GBP")
	log.Println("  turismo e PTAX:      USD-BRLT, EUR-BRLT, USD-BRLPTAX, EUR-BRLPTAX")
}

func fetchQuote(pair string) (*quoteResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), clientTimeout)
	defer cancel()

	reqURL := fmt.Sprintf("%s?moeda=%s", serverBaseURL, url.QueryEscape(pair))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("servidor retornou status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var quote quoteResponse
	if err := json.Unmarshal(body, &quote); err != nil {
		return nil, err
	}

	return &quote, nil
}

func saveToFile(label, bid string) error {
	content := fmt.Sprintf("%s: %s", label, bid)
	return os.WriteFile(outputFile, []byte(content), 0o644)
}
