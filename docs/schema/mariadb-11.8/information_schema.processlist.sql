-- 11.8.9-MariaDB-ubu2404-log
CREATE TEMPORARY TABLE `PROCESSLIST` (
  `ID` bigint(4) NOT NULL,
  `USER` varchar(128) NOT NULL,
  `HOST` varchar(64) NOT NULL,
  `DB` varchar(64),
  `COMMAND` varchar(16) NOT NULL,
  `TIME` int(7) NOT NULL,
  `STATE` varchar(64),
  `INFO` longtext,
  `TIME_MS` decimal(22,3) NOT NULL,
  `STAGE` tinyint(2) NOT NULL,
  `MAX_STAGE` tinyint(2) NOT NULL,
  `PROGRESS` decimal(7,3) NOT NULL,
  `MEMORY_USED` bigint(10) NOT NULL,
  `MAX_MEMORY_USED` bigint(10) NOT NULL,
  `EXAMINED_ROWS` bigint(10) NOT NULL,
  `SENT_ROWS` bigint(10) NOT NULL,
  `QUERY_ID` bigint(10) NOT NULL,
  `INFO_BINARY` blob,
  `TID` bigint(10) NOT NULL,
  `TMP_SPACE_USED` bigint(10) NOT NULL
) ENGINE=Aria DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_general_ci PAGE_CHECKSUM=0;
