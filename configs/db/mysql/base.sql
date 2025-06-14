CREATE TABLE `mcp_configs`
(
    `id`          BIGINT UNSIGNED     NOT NULL AUTO_INCREMENT,
    `name`        VARCHAR(50) NOT NULL,
    `tenant`      VARCHAR(50) NOT NULL DEFAULT '',
    `created_at`  DATETIME    NOT NULL,
    `updated_at`  DATETIME    NOT NULL,
    `routers`     TEXT,
    `servers`     TEXT,
    `tools`       TEXT,
    `mcp_servers` TEXT,
    `deleted_at`  DATETIME NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_name_tenant` (`tenant`, `name`),
    INDEX         `idx_mcp_configs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;


CREATE TABLE `mcp_config_versions`
(
    `id`          BIGINT UNSIGNED      NOT NULL AUTO_INCREMENT,
    `name`        VARCHAR(50)  NOT NULL,
    `tenant`      VARCHAR(50)  NOT NULL,
    `version`     INT          NOT NULL,
    `action_type` VARCHAR(50)  NOT NULL,
    `created_by`  VARCHAR(255),
    `created_at`  DATETIME     NOT NULL,
    `routers`     TEXT,
    `servers`     TEXT,
    `tools`       TEXT,
    `mcp_servers` TEXT,
    `hash`        VARCHAR(255) NOT NULL,
    `deleted_at`  DATETIME NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_name_tenant_version` (`name`, `tenant`, `version`),
    INDEX         `idx_mcp_config_versions_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `active_versions`
(
    `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `tenant`     VARCHAR(50) NOT NULL,
    `name`       VARCHAR(50) NOT NULL,
    `version`    INT         NOT NULL,
    `updated_at` DATETIME    NOT NULL,
    `deleted_at` DATETIME NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_tenant_name` (`tenant`, `name`),
    INDEX        `idx_active_versions_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;


CREATE TABLE `users`
(
    `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `username`   VARCHAR(50)  NOT NULL,
    `password`   VARCHAR(255) NOT NULL,
    `role`       VARCHAR(10)  NOT NULL DEFAULT 'normal',
    `is_active`  BOOLEAN      NOT NULL DEFAULT TRUE,
    `created_at` DATETIME     NOT NULL,
    `updated_at` DATETIME     NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_users_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `tenants`
(
    `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name`        VARCHAR(50) NOT NULL,
    `prefix`      VARCHAR(50) NOT NULL,
    `description` VARCHAR(255),
    `is_active`   BOOLEAN     NOT NULL DEFAULT TRUE,
    `created_at`  DATETIME    NOT NULL,
    `updated_at`  DATETIME    NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_tenants_name` (`name`),
    UNIQUE INDEX `idx_tenants_prefix` (`prefix`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `user_tenants`
(
    `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id`    BIGINT UNSIGNED NOT NULL,
    `tenant_id`  BIGINT UNSIGNED NOT NULL,
    `created_at` DATETIME NOT NULL,
    `updated_at` DATETIME NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_user_tenant` (`user_id`, `tenant_id`),
    CONSTRAINT `fk_user_tenants_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`),
    CONSTRAINT `fk_user_tenants_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Session 表创建语句
CREATE TABLE `sessions`
(
    `pk`         BIGINT NOT NULL AUTO_INCREMENT,
    `id`         VARCHAR(64)  DEFAULT NULL,
    `created_at` DATETIME     DEFAULT NULL,
    `title`      VARCHAR(255) DEFAULT NULL,
    PRIMARY KEY (`pk`),
    UNIQUE INDEX `idx_sessions_id` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Message 表创建语句
CREATE TABLE `messages`
(
    `pk`          BIGINT NOT NULL AUTO_INCREMENT,
    `id`          VARCHAR(64) DEFAULT NULL,
    `session_id`  VARCHAR(64) DEFAULT NULL,
    `content`     TEXT        DEFAULT NULL,
    `sender`      VARCHAR(50) DEFAULT NULL,
    `timestamp`   DATETIME    DEFAULT NULL,
    `tool_calls`  TEXT        DEFAULT NULL,
    `tool_result` TEXT        DEFAULT NULL,
    PRIMARY KEY (`pk`),
    UNIQUE INDEX `idx_messages_id` (`id`),
    INDEX         `idx_messages_session_id` (`session_id`),
    INDEX         `idx_messages_timestamp` (`timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;