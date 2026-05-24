# ⚡ Electricity Billing & Forecasting API

A robust RESTful API built with **Go 1.22+** and **PostgreSQL** to track, calculate, and forecast electricity consumption.

The API calculates precise bills using a cumulative slab-based tariff system (including special capacity rules), stores monthly meter readings, and utilizes statistical algorithms (Simple Moving Average, Weighted Moving Average, and Linear Regression) to forecast future energy usage and identify trends.

## ✨ Features

* **Go 1.22 Method-Based Routing:** Utilizes the new `http.NewServeMux()` routing enhancements.
* **Accurate Slab Calculation:** Dynamically calculates bills across multiple amperage capacities (5A, 15A, 30A, 60A) utilizing cumulative tariff logic.
* **Database Pooling:** High-performance database operations using `pgxpool`.
* **Predictive Analytics:** Built-in endpoints to forecast future consumption using mathematical models.
* **Graceful Shutdown:** Ensures zero data loss and safe database disconnection during server restarts.
* **CORS Ready:** Pre-configured for seamless integration with frontend frameworks like Vite/React (Port 5173).

---

## 🚀 Getting Started

### 1. Prerequisites

* [Go](https://go.dev/doc/install) (1.22 or higher)
* [PostgreSQL](https://www.postgresql.org/download/) (v12+)

### 2. Environment Setup

Create a `.env` file in the root of your project:

```env
DATABASE_URL=postgres://postgres:password@localhost:5432/electricity_bill_db
PORT=8080

```

### 3. Database Setup

Connect to your local Postgres instance and create the database:

```sql
CREATE DATABASE electricity_bill_db;

```

Run the following SQL commands to build your schema and seed the tariff rules:

```sql
CREATE TABLE MeterReadings (
    id SERIAL PRIMARY KEY,
    billing_month DATE NOT NULL,
    prev_reading_value DECIMAL(10, 2) NOT NULL,
    curr_reading_value DECIMAL(10, 2) NOT NULL,
    units_consumed DECIMAL(10, 2) NOT NULL,
    total_price DECIMAL(10, 2) NOT NULL,
    capacity_amps INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE TariffSlabs (
    id SERIAL PRIMARY KEY,
    capacity_amps INT NOT NULL,
    slab_start INT NOT NULL,
    slab_end INT,
    min_fee DECIMAL(10, 2) NOT NULL,
    rate_per_unit DECIMAL(10, 2) NOT NULL,
    special_rate_5a DECIMAL(10, 2)
);

CREATE INDEX idx_capacity ON TariffSlabs(capacity_amps);

-- Seed 5A Tariff Data
INSERT INTO TariffSlabs (capacity_amps, slab_start, slab_end, min_fee, rate_per_unit) VALUES
(5, 0, 20, 30.00, 0.00),
(5, 21, 30, 50.00, 6.50),
(5, 31, 50, 50.00, 8.00),
(5, 51, 100, 75.00, 9.50),
(5, 101, 250, 100.00, 9.50),
(5, 251, NULL, 150.00, 11.00);

-- Seed 15A Tariff Data
INSERT INTO TariffSlabs (capacity_amps, slab_start, slab_end, min_fee, rate_per_unit) VALUES
(15, 0, 20, 50.00, 4.00),
(15, 21, 30, 75.00, 6.50),
(15, 31, 50, 75.00, 8.00),
(15, 51, 100, 100.00, 9.50),
(15, 101, 250, 125.00, 9.50),
(15, 251, NULL, 175.00, 11.00);

-- Seed 30A Tariff Data
INSERT INTO TariffSlabs (capacity_amps, slab_start, slab_end, min_fee, rate_per_unit) VALUES
(30, 0, 20, 75.00, 5.00),
(30, 21, 30, 100.00, 6.50),
(30, 31, 50, 100.00, 8.00),
(30, 51, 100, 125.00, 9.50),
(30, 101, 250, 150.00, 9.50),
(30, 251, NULL, 200.00, 11.00);

-- Seed 60A Tariff Data
INSERT INTO TariffSlabs (capacity_amps, slab_start, slab_end, min_fee, rate_per_unit) VALUES
(60, 0, 20, 125.00, 6.00),
(60, 21, 30, 125.00, 6.50),
(60, 31, 50, 125.00, 8.00),
(60, 51, 100, 150.00, 9.50),
(60, 101, 250, 200.00, 9.50),
(60, 251, NULL, 250.00, 11.00);

```

### 4. Run the Application

Install dependencies and start the server:

```bash
go mod tidy
go run main.go

```

*The server will start on `http://localhost:8080`.*

---

## 📡 API Reference

### 1. Calculate Price

Calculates the expected bill on-the-fly without saving to the database.

* **URL:** `/api/calculate`
* **Method:** `GET`
* **Query Params:** * `units=[float]` (e.g., 100)
* `amps=[int]` (e.g., 5, 15, 30, 60)


* **Example Request:** `GET /api/calculate?units=100&amps=5`

### 2. Add Reading

Submits a new meter reading. The API automatically fetches the previous month's reading, calculates the consumed units, computes the bill, and saves the record.

* **URL:** `/api/readings`
* **Method:** `POST`
* **Body (`application/json`):**
```json
{
  "current_value": 550,
  "amps": 5,
  "month": "2024-07-01"
}

```



### 3. Get Readings

Retrieves the complete history of meter readings and billing statements, sorted by the most recent month.

* **URL:** `/api/readings`
* **Method:** `GET`
* **Success Response:** Array of reading objects detailing units used, capacities, and costs.

### 4. Get Forecast

Analyzes the last 6 months of database records to predict future consumption and analyze usage trends.

* **URL:** `/api/forecast`
* **Method:** `GET`
* **Success Response:**
```json
{
  "forecast": {
    "sma_3_month": 110.5,
    "trend_forecast": 112.3,
    "wma_3_month": 115.1
  },
  "history": [90, 100, 110, 105, 120, 115],
  "trend_direction": "Usage is trending UP. Consider energy-saving measures."
}

```



---

## 🧮 Forecasting Algorithms Used

The `/api/forecast` endpoint uses three mathematical models to analyze consumption data.

**1. Simple Moving Average (SMA)**
Calculates the unweighted mean of the previous $n$ data points.


$$SMA = \frac{1}{n} \sum_{i=1}^{n} P_i$$

**2. Weighted Moving Average (WMA)**
Assigns a heavier weighting to more recent data points to make the prediction more responsive to recent behavior (uses weights: 0.2, 0.3, 0.5).


$$WMA = \sum_{i=1}^{n} (P_i \times W_i)$$

**3. Linear Regression**
Fits a straight line through the historical data points to predict the next sequential value ($y = mx + c$).

* **Slope ($m$):**

$$m = \frac{n(\sum xy) - (\sum x)(\sum y)}{n(\sum x^2) - (\sum x)^2}$$


* **Intercept ($c$):**

$$c = \frac{\sum y - m(\sum x)}{n}$$