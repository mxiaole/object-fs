CREATE TABLE `img`
(
    `id`        int                                                           NOT NULL AUTO_INCREMENT,
    `bucket_id` int                                                           NOT NULL,
    `name`      varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
    `data`      longblob                                                      NOT NULL,
    `md5`       varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_md5` (`md5`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;