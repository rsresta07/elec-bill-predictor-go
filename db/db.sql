-- Track individual meter readings over time
CREATE TABLE MeterReadings (
    id SERIAL PRIMARY KEY,
    reading_date DATE NOT NULL UNIQUE,
    kwh_consumed DECIMAL(10, 2) NOT NULL
);

-- Store the tariff logic so you can change rates without redeploying code
CREATE TABLE TariffSlabs (
    id SERIAL PRIMARY KEY,
    capacity_amps INT NOT NULL,     -- 5, 15, 30, 60
    slab_start INT NOT NULL,        -- 0, 21, 31, 51, 101, 251
    slab_end INT,                   -- NULL for 251+
    min_fee DECIMAL(10, 2) NOT NULL,
    rate_per_unit DECIMAL(10, 2) NOT NULL,
    special_rate_5a DECIMAL(10, 2)  -- To handle that "Rs 3 if > 20 units" rule
);

-- Indexing for faster lookups
CREATE INDEX idx_capacity ON TariffSlabs(capacity_amps);