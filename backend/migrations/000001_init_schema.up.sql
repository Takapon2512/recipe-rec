-- users
CREATE TABLE users (
  id           BIGINT       NOT NULL AUTO_INCREMENT,
  cognito_sub  VARCHAR(64)  NOT NULL,
  email        VARCHAR(255) NOT NULL,
  display_name VARCHAR(100) DEFAULT NULL,
  provider     VARCHAR(20)  NOT NULL,
  created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at   DATETIME     DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_cognito_sub (cognito_sub),
  KEY idx_email (email),
  KEY idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- categories
CREATE TABLE categories (
  id                       INT         NOT NULL AUTO_INCREMENT,
  name                     VARCHAR(50) NOT NULL,
  type                     VARCHAR(20) NOT NULL,
  default_storage_location VARCHAR(20) DEFAULT NULL,
  created_at               DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at               DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at               DATETIME    DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- inventory_items
CREATE TABLE inventory_items (
  id               BIGINT        NOT NULL AUTO_INCREMENT,
  user_id          BIGINT        NOT NULL,
  category_id      INT           DEFAULT NULL,
  name             VARCHAR(100)  NOT NULL,
  quantity         DECIMAL(10,2) NOT NULL,
  unit             VARCHAR(20)   NOT NULL,
  purchased_at     DATE          DEFAULT NULL,
  expires_at       DATE          DEFAULT NULL,
  storage_location VARCHAR(20)   DEFAULT NULL,
  memo             TEXT          DEFAULT NULL,
  created_at       DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at       DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at       DATETIME      DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_user_id (user_id),
  KEY idx_user_expires (user_id, expires_at),
  KEY idx_deleted_at (deleted_at),
  CONSTRAINT fk_inv_user FOREIGN KEY (user_id)     REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_inv_cat FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- recipes
CREATE TABLE recipes (
  id               BIGINT       NOT NULL AUTO_INCREMENT,
  user_id          BIGINT       NOT NULL,
  title            VARCHAR(200) NOT NULL,
  cooking_time_min INT          DEFAULT NULL,
  genre            VARCHAR(20)  DEFAULT NULL,
  source           VARCHAR(20)  NOT NULL,
  created_at       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at       DATETIME     DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_user_id (user_id),
  KEY idx_deleted_at (deleted_at),
  CONSTRAINT fk_recipe_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- recipe_ingredients
CREATE TABLE recipe_ingredients (
  id         BIGINT        NOT NULL AUTO_INCREMENT,
  recipe_id  BIGINT        NOT NULL,
  name       VARCHAR(100)  NOT NULL,
  quantity   DECIMAL(10,2) DEFAULT NULL,
  unit       VARCHAR(20)   DEFAULT NULL,
  created_at DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at DATETIME      DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_recipe_id (recipe_id),
  CONSTRAINT fk_ing_recipe FOREIGN KEY (recipe_id) REFERENCES recipes(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- recipe_steps
CREATE TABLE recipe_steps (
  id           BIGINT   NOT NULL AUTO_INCREMENT,
  recipe_id    BIGINT   NOT NULL,
  step_no      INT      NOT NULL,
  description  TEXT     NOT NULL,
  duration_min INT      DEFAULT NULL,
  created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at   DATETIME DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_recipe_step_active (recipe_id, step_no, (IFNULL(deleted_at, '9999-12-31'))),
  CONSTRAINT fk_step_recipe FOREIGN KEY (recipe_id) REFERENCES recipes(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- meal_plans
CREATE TABLE meal_plans (
  id         BIGINT       NOT NULL AUTO_INCREMENT,
  user_id    BIGINT       NOT NULL,
  plan_date  DATE         NOT NULL,
  meal_type  VARCHAR(20)  NOT NULL,
  dish_name  VARCHAR(200) NOT NULL,
  recipe_id  BIGINT       DEFAULT NULL,
  memo       TEXT         DEFAULT NULL,
  created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at DATETIME     DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_user_date (user_id, plan_date),
  KEY idx_deleted_at (deleted_at),
  CONSTRAINT fk_mp_user   FOREIGN KEY (user_id)   REFERENCES users(id)   ON DELETE CASCADE,
  CONSTRAINT fk_mp_recipe FOREIGN KEY (recipe_id) REFERENCES recipes(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- recommendation_logs
CREATE TABLE recommendation_logs (
  id               BIGINT       NOT NULL AUTO_INCREMENT,
  user_id          BIGINT       NOT NULL,
  status           VARCHAR(20)  NOT NULL DEFAULT 'pending',
  request_payload  JSON         NOT NULL,
  response_payload JSON         DEFAULT NULL,
  error_code       VARCHAR(50)  DEFAULT NULL,
  error_message    TEXT         DEFAULT NULL,
  latency_ms       INT          DEFAULT NULL,
  created_at       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at       DATETIME     DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_user_created (user_id, created_at),
  KEY idx_status_created (status, created_at),
  CONSTRAINT fk_log_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
