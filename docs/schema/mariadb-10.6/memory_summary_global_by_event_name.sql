-- 10.6.28-MariaDB-ubu2204-log
CREATE TABLE `memory_summary_global_by_event_name` (
  `EVENT_NAME` varchar(128) NOT NULL COMMENT 'Event name.',
  `COUNT_ALLOC` bigint(20) unsigned NOT NULL COMMENT 'Total number of allocations to memory.',
  `COUNT_FREE` bigint(20) unsigned NOT NULL COMMENT 'Total number of attempts to free the allocated memory.',
  `SUM_NUMBER_OF_BYTES_ALLOC` bigint(20) unsigned NOT NULL COMMENT 'Total number of bytes allocated.',
  `SUM_NUMBER_OF_BYTES_FREE` bigint(20) unsigned NOT NULL COMMENT 'Total number of bytes freed',
  `LOW_COUNT_USED` bigint(20) NOT NULL COMMENT 'Lowest number of allocated blocks (lowest value of CURRENT_COUNT_USED).',
  `CURRENT_COUNT_USED` bigint(20) NOT NULL COMMENT 'Currently allocated blocks that have not been freed (COUNT_ALLOC minus COUNT_FREE).',
  `HIGH_COUNT_USED` bigint(20) NOT NULL COMMENT 'Highest number of allocated blocks (highest value of CURRENT_COUNT_USED).',
  `LOW_NUMBER_OF_BYTES_USED` bigint(20) NOT NULL COMMENT 'Lowest number of bytes used.',
  `CURRENT_NUMBER_OF_BYTES_USED` bigint(20) NOT NULL COMMENT 'Current number of bytes used (total allocated minus total freed).',
  `HIGH_NUMBER_OF_BYTES_USED` bigint(20) NOT NULL COMMENT 'Highest number of bytes used.'
) ENGINE=PERFORMANCE_SCHEMA DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_general_ci;
