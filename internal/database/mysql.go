// Package database owns the MySQL (GORM) and Redis connections plus schema
// migrations, as package-level singletons initialized once at startup.
package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"excel-crud-app/internal/config"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the shared GORM handle every service uses to talk to MySQL.
var DB *gorm.DB

// ConnectMySQL creates a MySQL DB connection and verifies connectivity.
func ConnectMySQL(cfg *config.Config) error {

	// Connect WITHOUT database
	serverDSN := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/",
		cfg.MySQLUser,
		cfg.MySQLPassword,
		cfg.MySQLHost,
		cfg.MySQLPort,
	)

	sqlDB, err := sql.Open("mysql", serverDSN)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	// Create database if it doesn't exist
	_, err = sqlDB.Exec(fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		cfg.MySQLDBName,
	))
	if err != nil {
		return err
	}

	// Connect TO database
	dbDSN := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.MySQLUser,
		cfg.MySQLPassword,
		cfg.MySQLHost,
		cfg.MySQLPort,
		cfg.MySQLDBName,
	)

	// logger.Warn: StructuredLogger already covers request-level logging.
	db, err := gorm.Open(mysql.Open(dbDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return err
	}

	gormDB, err := db.DB()
	if err != nil {
		return err
	}

	// Conservative pool sizing for a single-instance deployment.
	gormDB.SetMaxOpenConns(25)
	gormDB.SetMaxIdleConns(10)
	gormDB.SetConnMaxLifetime(5 * time.Minute)

	DB = db

	log.Println("Connected to MySQL successfully")

	return nil
}