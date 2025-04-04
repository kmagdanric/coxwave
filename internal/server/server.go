package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"

	"coupons/proto/couponpbconnect"
)

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// Server encapsulates the coupon service
type Server struct {
	couponpbconnect.UnimplementedCouponServiceHandler
	db          *sql.DB
	redisClient *redis.Client
	mu          sync.Mutex
	mux         *http.ServeMux
}

// New creates a new Server instance
func New() (*Server, error) {
	// Get environment variables with defaults
	dbHost := getEnv("DB_HOST", "127.0.0.1")
	dbPort := getEnv("DB_PORT", "3306")
	dbUser := getEnv("DB_USER", "coxwave")
	dbPassword := getEnv("DB_PASSWORD", "coxwavewave")
	dbName := getEnv("DB_NAME", "coupons")
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")

	// Setup MySQL connection
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", 
		dbUser, dbPassword, dbHost, dbPort, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(50)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Setup Redis client
	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort)
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		PoolSize: 100,
		MinIdleConns: 50,
	})

	s := &Server{
		db:          db,
		redisClient: redisClient,
		mux:         http.NewServeMux(),
	}

	path, handler := couponpbconnect.NewCouponServiceHandler(s)
	s.mux.Handle(path, handler)

	return s, nil
}

// Handler returns the HTTP handler for the server
func (s *Server) Handler() http.Handler {
	return s.mux
}

// Close closes all connections
func (s *Server) Close() error {
	s.db.Close()
	s.redisClient.Close()
	return nil
}
