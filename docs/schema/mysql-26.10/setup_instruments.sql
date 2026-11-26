-- 26.10.0
CREATE TABLE `setup_instruments` (
  `NAME` varchar(128) NOT NULL,
  `ENABLED` enum('YES','NO') NOT NULL,
  `TIMED` enum('YES','NO') DEFAULT NULL,
  `PROPERTIES` set('singleton','progress','user','global_statistics','mutable','controlled_by_default') NOT NULL,
  `FLAGS` set('controlled') DEFAULT NULL,
  `VOLATILITY` int NOT NULL,
  `DOCUMENTATION` longtext,
  PRIMARY KEY (`NAME`)
) ENGINE=PERFORMANCE_SCHEMA DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
