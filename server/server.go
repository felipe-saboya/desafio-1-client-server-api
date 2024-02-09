package main

import (
	"context"
	"encoding/json"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	serverPort = ":8080"
	dbPath     = "../data/Rates.db"
	timeoutAPI = 200 * time.Millisecond
	timeoutDB  = 10 * time.Millisecond
)

type DollarRealRate struct {
	UsdBrl struct {
		Code       string  `json:"code"`
		CodeIn     string  `json:"codein"`
		Name       string  `json:"name"`
		High       float64 `json:"high,string"`
		Low        float64 `json:"low,string"`
		VarBid     float64 `json:"varBid,string"`
		PctChange  float64 `json:"pctChange,string"`
		Bid        float64 `json:"bid,string"`
		Ask        float64 `json:"ask,string"`
		Timestamp  string  `json:"timestamp"`
		CreateDate string  `json:"create_date"`
	} `json:"USDBRL"`
}

type RateBid struct {
	Bid float64 `json:"bid"`
}

func main() {
	// 1. Start database
	db := initDB()

	// 2. Handler
	http.HandleFunc("/cotacao", func(w http.ResponseWriter, r *http.Request) {
		// 2.1 New context
		ctx, cancel := context.WithTimeout(r.Context(), timeoutAPI)
		defer cancel()

		// 2.2 Get Rate
		rate, err := getDollarRealRate(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 2.3 Return only rate Bid
		if err := json.NewEncoder(w).Encode(RateBid{Bid: rate.UsdBrl.Bid}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 2.4 Save client search in database
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), timeoutDB)
			defer cancel()

			if err := saveRate(ctx, db, rate); err != nil {
				log.Println("Error saving dollarRealRate to DB:", err)
			}
		}()
	})

	log.Fatal(http.ListenAndServe(serverPort, nil))
}

// 1. Start database
func initDB() *sqlx.DB {
	db, err := sqlx.Connect("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS USD_BRL (
                           id INTEGER PRIMARY KEY AUTOINCREMENT,
                           code TEXT,
                           code_in TEXT,
                           name TEXT,
                           high REAL,
                           low REAL,
                           var_bid REAL,
                           pct_change REAL,
                           bid REAL,
                           ask REAL,
                           timestamp TEXT,
                           create_date TEXT,
                           search_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                         )`); err != nil {
		log.Fatal("Error creating table:", err)
	}

	return db
}

// 2.2 Get Rate
func getDollarRealRate(ctx context.Context) (DollarRealRate, error) {
	client := http.Client{Timeout: timeoutAPI}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return DollarRealRate{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return DollarRealRate{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("Error on close connection:", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return DollarRealRate{}, err
	}

	rate := DollarRealRate{}

	if err := json.Unmarshal(body, &rate); err != nil {
		return DollarRealRate{}, err
	}

	return rate, nil
}

// 2.4 Save client search in database
func saveRate(ctx context.Context, db *sqlx.DB, rate DollarRealRate) error {
	var _, err = db.ExecContext(ctx,
		`INSERT INTO USD_BRL (code, code_in, name, high, low, var_bid, pct_change, bid, ask, timestamp, create_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rate.UsdBrl.Code,
		rate.UsdBrl.CodeIn,
		rate.UsdBrl.Name,
		rate.UsdBrl.High,
		rate.UsdBrl.Low,
		rate.UsdBrl.VarBid,
		rate.UsdBrl.PctChange,
		rate.UsdBrl.Bid,
		rate.UsdBrl.Ask,
		rate.UsdBrl.Timestamp,
		rate.UsdBrl.CreateDate)
	return err
}
