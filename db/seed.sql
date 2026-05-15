-- Insert 5A Tariff Data
INSERT INTO TariffSlabs (capacity_amps, slab_start, slab_end, min_fee, rate_per_unit) VALUES
(5, 0, 20, 30.00, 0.00),   -- Special logic in code handles the Rs. 3 jump
(5, 21, 30, 50.00, 6.50),
(5, 31, 50, 50.00, 8.00),
(5, 51, 100, 75.00, 9.50),
(5, 101, 250, 100.00, 9.50),
(5, 251, NULL, 150.00, 11.00);

-- Insert 15A Tariff Data
INSERT INTO TariffSlabs (capacity_amps, slab_start, slab_end, min_fee, rate_per_unit) VALUES
(15, 0, 20, 50.00, 4.00),
(15, 21, 30, 75.00, 6.50),
(15, 31, 50, 75.00, 8.00),
(15, 51, 100, 100.00, 9.50),
(15, 101, 250, 125.00, 9.50),
(15, 251, NULL, 175.00, 11.00);

-- Insert 30A Tariff Data
INSERT INTO TariffSlabs (capacity_amps, slab_start, slab_end, min_fee, rate_per_unit) VALUES
(30, 0, 20, 75.00, 5.00),
(30, 21, 30, 100.00, 6.50),
(30, 31, 50, 100.00, 8.00),
(30, 51, 100, 125.00, 9.50),
(30, 101, 250, 150.00, 9.50),
(30, 251, NULL, 200.00, 11.00);