package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"electricity-api/db"
	"electricity-api/electricity"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var pool *pgxpool.Pool

func main() {
	// 1. Load Environment Variables
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, relying on system environment variables")
	}

	// 2. Connect to Database Pool
	var err error
	pool, err = db.Connect()
	if err != nil {
		log.Fatalf("❌ Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	// 3. Set up Router utilizing Go 1.22+ method-based routing
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/calculate", handleCalculate)
	mux.HandleFunc("POST /api/readings", handlePostReading)
	mux.HandleFunc("GET /api/readings", handleGetReadings)
	mux.HandleFunc("GET /api/forecast", handleDBForecast)

	// 4. Wrap router with CORS middleware
	handler := corsMiddleware(mux)

	// 5. Configure Server with Graceful Shutdown
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	go func() {
		fmt.Printf("⚡ Electricity API is live at http://localhost:%s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown listener
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}

// --- MIDDLEWARE ---

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}


func handleCalculate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	units, _ := strconv.ParseFloat(r.URL.Query().Get("units"), 64)
	amps, _ := strconv.Atoi(r.URL.Query().Get("amps"))

	slabs, err := fetchSlabs(ctx, amps)
	if err != nil {
		http.Error(w, `{"error": "Error fetching tariff data"}`, http.StatusInternalServerError)
		return
	}

	report := electricity.CalculateExactBill(amps, units, slabs)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func handlePostReading(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var input struct {
		CurrentValue float64 `json:"current_value"`
		Amps         int     `json:"amps"`
		Month        string  `json:"month"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error": "Invalid input JSON"}`, http.StatusBadRequest)
		return
	}

	var prevValue float64
	err := pool.QueryRow(ctx, "SELECT curr_reading_value FROM MeterReadings ORDER BY billing_month DESC LIMIT 1").Scan(&prevValue)
	if err != nil {
		prevValue = input.CurrentValue // Initial baseline if no history
	}

	unitsConsumed := input.CurrentValue - prevValue
	if unitsConsumed < 0 {
		http.Error(w, `{"error": "Current reading cannot be lower than previous"}`, http.StatusBadRequest)
		return
	}

	slabs, _ := fetchSlabs(ctx, input.Amps)
	billReport := electricity.CalculateExactBill(input.Amps, unitsConsumed, slabs)

	_, err = pool.Exec(ctx, `
      INSERT INTO MeterReadings 
      (billing_month, prev_reading_value, curr_reading_value, units_consumed, total_price, capacity_amps) 
      VALUES ($1, $2, $3, $4, $5, $6)`,
		input.Month, prevValue, input.CurrentValue, unitsConsumed, billReport.TotalAmount, input.Amps)

	if err != nil {
		http.Error(w, `{"error": "Database save error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func handleGetReadings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := pool.Query(ctx, `
        SELECT 
            billing_month, 
            COALESCE(prev_reading_value, 0), 
            COALESCE(curr_reading_value, 0), 
            COALESCE(units_consumed, 0), 
            COALESCE(total_price, 0), 
            COALESCE(capacity_amps, 0) 
        FROM MeterReadings 
        ORDER BY billing_month DESC`)

	if err != nil {
		http.Error(w, `{"error": "Database query error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var month time.Time
		var prev, curr, units, price float64
		var amps int

		if err := rows.Scan(&month, &prev, &curr, &units, &price, &amps); err != nil {
			continue
		}

		history = append(history, map[string]interface{}{
			"month":          month.Format("2006-01-02"),
			"previous_meter": prev,
			"current_meter":  curr,
			"units_used":     units,
			"cost":           price,
			"capacity":       amps,
		})
	}

	// Prevent returning null if array is empty
	if history == nil {
		history = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func handleDBForecast(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := pool.Query(ctx, "SELECT units_consumed FROM MeterReadings ORDER BY billing_month DESC LIMIT 6")
	if err != nil {
		http.Error(w, `{"error": "Database query error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var history []float64
	for rows.Next() {
		var val float64
		rows.Scan(&val)
		history = append(history, val)
	}

	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	recentHistory := history
	if len(history) > 3 {
		recentHistory = history[len(history)-3:]
	}
	sma := electricity.CalculateSMA(recentHistory)
	wma := electricity.CalculateWMA(recentHistory, []float64{0.2, 0.3, 0.5})
	linReg := electricity.CalculateLinearRegression(history)

	response := map[string]interface{}{
		"history": history,
		"forecast": map[string]interface{}{
			"sma_3_month":    math.Round(sma*100) / 100,
			"wma_3_month":    math.Round(wma*100) / 100,
			"trend_forecast": math.Round(linReg*100) / 100,
		},
		"trend_direction": identifyTrend(linReg, sma),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func identifyTrend(reg float64, sma float64) string {
	if reg > sma+5 {
		return "Usage is trending UP. Consider energy-saving measures."
	} else if reg < sma-5 {
		return "Usage is trending DOWN. Great job!"
	}
	return "Usage is stable."
}

func fetchSlabs(ctx context.Context, amps int) ([]electricity.Slab, error) {
	rows, err := pool.Query(ctx,
		"SELECT slab_start, COALESCE(slab_end, 0), min_fee, rate_per_unit FROM TariffSlabs WHERE capacity_amps=$1 ORDER BY slab_start ASC",
		amps)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slabs []electricity.Slab
	for rows.Next() {
		var s electricity.Slab
		if err := rows.Scan(&s.Start, &s.End, &s.MinFee, &s.Rate); err != nil {
			return nil, err
		}
		slabs = append(slabs, s)
	}
	return slabs, nil
}
