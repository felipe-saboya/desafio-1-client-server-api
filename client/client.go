package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	serverURL   = "http://localhost:8080/cotacao"
	timeoutHTTP = 300 * time.Millisecond
)

type RateBid struct {
	Bid float64 `json:"bid"`
}

func main() {
	// 1. Start context
	ctx, cancel := context.WithTimeout(context.Background(), timeoutHTTP)
	defer cancel()

	// 2. Get rate from server
	rate, err := getRate(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// 3. Save txt file
	if err := saveRate(rate.Bid); err != nil {
		log.Fatal("Error saving rate to file:", err)
	}
}

// 2. Get rate from server
func getRate(ctx context.Context) (RateBid, error) {
	client := http.Client{Timeout: timeoutHTTP}
	req, err := http.NewRequestWithContext(ctx, "GET", serverURL, nil)
	if err != nil {
		return RateBid{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return RateBid{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("Error on close connection:", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return RateBid{}, fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	rate := RateBid{}
	if err := json.NewDecoder(resp.Body).Decode(&rate); err != nil {
		return RateBid{}, err
	}

	return rate, nil
}

// 3. Save txt file
func saveRate(bid float64) error {
	data := []byte(fmt.Sprintf("Dólar: %.2f", bid))
	return os.WriteFile("cotacao.txt", data, 0644)
}
