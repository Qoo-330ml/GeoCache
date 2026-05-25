package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openDatabase(path string) (*gorm.DB, error) {
	if path == "" {
		path = "/data/license.db"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}
	db, err := gorm.Open(sqlite.Open(path+"?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	if err := db.AutoMigrate(&ActivationCode{}, &LicenseCheck{}, &ClientInstall{}, &IPReport{}, &IPBest{}, &FeaturePolicy{}); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	if err := seedFeaturePolicies(db); err != nil {
		return nil, err
	}
	return db, nil
}

func seedFeaturePolicies(db *gorm.DB) error {
	for _, policy := range defaultFeaturePolicies {
		var count int64
		if err := db.Model(&FeaturePolicy{}).Where("key = ?", policy.Key).Count(&count).Error; err != nil {
			return fmt.Errorf("check feature policy %s: %w", policy.Key, err)
		}
		if count > 0 {
			continue
		}
		if err := db.Create(&policy).Error; err != nil {
			return fmt.Errorf("seed feature policy %s: %w", policy.Key, err)
		}
	}
	return nil
}
