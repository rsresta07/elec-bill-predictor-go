package main

import (
	"context"
	"electricity-api/db"
	"electricity-api/electricity"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

// Global database connection
var conn *pgx.Conn

func main() {
	var err error
	// Ensure you have set: export DATABASE_URL=postgres://user:pass@localhost:5432/dbname
	conn, err = db.Connect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	// --- ROUTES ---

	// GET: Calculate a bill on the fly (e.g., /api/calculate?units=100&amps=5)
	http.HandleFunc("/api/calculate", handleCalculate)

	// POST: Save a new meter reading to DB
	// GET: List all historical readings
	http.HandleFunc("/api/readings", handleReadings)

	// GET: Forecast next month based on DB history
	http.HandleFunc("/api/forecast", handleDBForecast)

	fmt.Println("⚡ Electricity API is live at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// --- HANDLERS ---

// handleCalculate handles real-time calculation requests
func handleCalculate(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	unitsStr := r.URL.Query().Get("units")
	ampsStr := r.URL.Query().Get("amps")

	units, _ := strconv.ParseFloat(unitsStr, 64)
	amps, _ := strconv.Atoi(ampsStr)

	slabs, err := fetchSlabs(ctx, amps)
	if err != nil {
		http.Error(w, "Error fetching tariff data", 500)
		return
	}

	report := electricity.CalculateExactBill(amps, units, slabs)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// handleReadings saves or retrieves readings from the Postgres DB
func handleReadings(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	if r.Method == http.MethodPost {
		var input struct {
			Units float64 `json:"units"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid input", 400)
			return
		}

		_, err := conn.Exec(ctx,
			"INSERT INTO MeterReadings (reading_date, kwh_consumed) VALUES ($1, $2)",
			time.Now(), input.Units)

		if err != nil {
			http.Error(w, "DB Error: "+err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, "Reading saved successfully")

	} else if r.Method == http.MethodGet {
		rows, _ := conn.Query(ctx, "SELECT reading_date, kwh_consumed FROM MeterReadings ORDER BY reading_date DESC")
		var readings []map[string]interface{}

		for rows.Next() {
			var d time.Time
			var k float64
			rows.Scan(&d, &k)
			readings = append(readings, map[string]interface{}{
				"date": d.Format("2006-01-02"),
				"kwh":  k,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(readings)
	}
}

// handleDBForecast pulls the last 3 months and predicts the 4th
func handleDBForecast(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Get last 3 readings
	rows, err := conn.Query(ctx, "SELECT kwh_consumed FROM MeterReadings ORDER BY reading_date DESC LIMIT 3")
	if err != nil {
		http.Error(w, "DB Error", 500)
		return
	}

	var history []float64
	for rows.Next() {
		var val float64
		rows.Scan(&val)
		history = append(history, val)
	}

	if len(history) < 3 {
		http.Error(w, "Need at least 3 months of history to forecast", 400)
		return
	}

	// Flip history to chronological order [oldest, ..., newest]
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	sma := electricity.CalculateSMA(history)
	wma := electricity.CalculateWMA(history, []float64{0.2, 0.3, 0.5})

	response := map[string]interface{}{
		"history": history,
		"forecast": map[string]float64{
			"sma_prediction": sma,
			"wma_prediction": wma,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// --- HELPERS ---

// fetchSlabs grabs the pricing tiers from Postgres
func fetchSlabs(ctx context.Context, amps int) ([]electricity.Slab, error) {
	rows, err := conn.Query(ctx,
		"SELECT slab_start, COALESCE(slab_end, 0), min_fee, rate_per_unit FROM TariffSlabs WHERE capacity_amps=$1 ORDER BY slab_start ASC",
		amps)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slabs []electricity.Slab
	for rows.Next() {
		var s electricity.Slab
		err := rows.Scan(&s.Start, &s.End, &s.MinFee, &s.Rate)
		if err != nil {
			return nil, err
		}
		slabs = append(slabs, s)
	}
	return slabs, nil
}
