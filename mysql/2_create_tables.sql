use coupons;

-- Table to store coupon campaigns.
CREATE TABLE campaigns (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    start_time DATETIME NOT NULL,
    total_coupons BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- Table to store issued coupons.
CREATE TABLE coupons (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    campaign_id BIGINT NOT NULL,
    coupon_code VARCHAR(15) NOT NULL,
    issued_at DATETIME NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id)
        ON DELETE CASCADE
) ENGINE=InnoDB;

-- Ensure coupon codes are unique.
CREATE UNIQUE INDEX idx_coupon_code ON coupons(coupon_code);
