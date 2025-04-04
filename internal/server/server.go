package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"google.golang.org/protobuf/types/known/timestamppb"
	couponpb "coupons/proto"
	"coupons/proto/couponpbconnect"
)

type Campaign struct {
	ID           int64
	Name         string
	StartTime    time.Time
	TotalCoupons int64
}

type CouponService struct {
	couponpbconnect.UnimplementedCouponServiceHandler
	db          *sql.DB
	redisClient *redis.Client
	mu          sync.Mutex
}

func NewCouponService(db *sql.DB, redisClient *redis.Client) *CouponService {
	return &CouponService{
		db:          db,
		redisClient: redisClient,
	}
}

func (s *CouponService) CreateCampaign(
	ctx context.Context,
	req *connect.Request[couponpb.CreateCampaignRequest],
) (*connect.Response[couponpb.CreateCampaignResponse], error) {
	campaign := req.Msg
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO campaigns(name, start_time, total_coupons) VALUES (?, ?, ?)",
		campaign.Name, campaign.StartTime.AsTime(), campaign.TotalCoupons)
	if err != nil {
		return nil, fmt.Errorf("failed to insert campaign: %w", err)
	}
	campaignID, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get campaign id: %w", err)
	}

	redisKey := fmt.Sprintf("campaign:%d:startTime", campaignID)
	err = s.redisClient.Set(ctx, redisKey, campaign.StartTime.AsTime(), 0).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to set redis campaign start time: %w", err)
	}

	redisKey = fmt.Sprintf("campaign:%d:coupons", campaignID)
	err = s.redisClient.Set(ctx, redisKey, campaign.TotalCoupons, 0).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to set redis counter: %w", err)
	}

	resp := &couponpb.CreateCampaignResponse{
		CampaignId: campaignID,
	}
	return connect.NewResponse(resp), nil
}

var koreanSyllables = []rune{
	'가', '나', '다', '라', '마', '바', '사', '아', '자', '차', '카', '타', '파', '하',
	'개', '내', '대', '래', '매', '배', '새', '애', '재', '채', '캐', '태', '패', '해',
	'고', '노', '도', '로', '모', '보', '소', '오', '조', '초', '코', '토', '포', '호',
}

// Format: KKNNNNKKNN (K: Korean character, N: number)
func generateCouponCode(campaignID int64) string {
	// Create a unique seed combining:
	// 1. Campaign ID
	// 2. Current nano time
	// 3. Random bytes from crypto/rand
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to timestamp-based randomness if crypto/rand fails
		randomBytes = []byte(fmt.Sprintf("%d", timestamp))
	}

	data := make([]byte, 24) // Increased size for more entropy
	binary.BigEndian.PutUint64(data[0:8], uint64(campaignID))
	binary.BigEndian.PutUint64(data[8:16], uint64(timestamp))
	copy(data[16:], randomBytes)
	
	hash := sha256.Sum256(data)
	
	var sb strings.Builder

	// Use crypto/rand for Korean characters
	for i := 0; i < 2; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(koreanSyllables))))
		if err != nil {
			// Fallback to hash-based selection
			idx = big.NewInt(int64(hash[i] % uint8(len(koreanSyllables))))
		}
		sb.WriteRune(koreanSyllables[idx.Int64()])
	}

	// Use crypto/rand for numbers
	for i := 0; i < 4; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			// Fallback to hash-based selection
			num = big.NewInt(int64(hash[i] % 10))
		}
		sb.WriteString(fmt.Sprintf("%d", num))
	}

	// Use crypto/rand for Korean characters
	for i := 0; i < 2; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(koreanSyllables))))
		if err != nil {
			// Fallback to hash-based selection
			idx = big.NewInt(int64(hash[i+4] % uint8(len(koreanSyllables))))
		}
		sb.WriteRune(koreanSyllables[idx.Int64()])
	}

	// Use crypto/rand for numbers
	for i := 0; i < 2; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			// Fallback to hash-based selection
			num = big.NewInt(int64(hash[i+6] % 10))
		}
		sb.WriteString(fmt.Sprintf("%d", num))
	}

	return sb.String()
}

func (s *CouponService) IssueCoupon(
	ctx context.Context,
	req *connect.Request[couponpb.IssueCouponRequest],
) (*connect.Response[couponpb.IssueCouponResponse], error) {
	campaignID := req.Msg.CampaignId

	redisStartTimeKey := fmt.Sprintf("campaign:%d:startTime", campaignID)
	startTimeStr, err := s.redisClient.Get(ctx, redisStartTimeKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get campaign start time: %w", err)
	}
	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid start time format in redis: %w", err)
	}
	if time.Now().Before(startTime) {
		return nil, fmt.Errorf("coupon issuance not started yet")
	}

	redisKey := fmt.Sprintf("campaign:%d:coupons", campaignID)
	val, err := s.redisClient.Decr(ctx, redisKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to decrement redis counter: %w", err)
	}

	if val < 0 {
		s.redisClient.Incr(ctx, redisKey)
		return nil, fmt.Errorf("no more coupons available")
	}

	couponCode := generateCouponCode(campaignID)
	saveCoupon := "INSERT INTO coupons(campaign_id, coupon_code, issued_at) VALUES (?, ?, ?)"
	_, err = s.db.ExecContext(ctx, saveCoupon, campaignID, couponCode, time.Now())
	if err != nil {
		s.redisClient.Incr(ctx, redisKey)
		return nil, fmt.Errorf("failed to persist coupon issuance: %w", err)
	}

	resp := &couponpb.IssueCouponResponse{
		CouponCode: couponCode,
	}
	return connect.NewResponse(resp), nil
}

func (s *CouponService) GetCampaign(
	ctx context.Context,
	req *connect.Request[couponpb.GetCampaignRequest],
) (*connect.Response[couponpb.GetCampaignResponse], error) {
	campaignID := req.Msg.CampaignId

	var campaign Campaign
	selectCampaign := "SELECT id, name, start_time, total_coupons FROM campaigns WHERE id = ?"
	err := s.db.QueryRowContext(ctx, selectCampaign, campaignID).
		Scan(&campaign.ID, &campaign.Name, &campaign.StartTime, &campaign.TotalCoupons)
	if err != nil {
		return nil, fmt.Errorf("campaign not found: %w", err)
	}

	selectCoupons := "SELECT coupon_code FROM coupons WHERE campaign_id = ?"
	rows, err := s.db.QueryContext(ctx, selectCoupons, campaignID)
	if err != nil {
		return nil, fmt.Errorf("failed to query coupons: %w", err)
	}
	defer rows.Close()
	var coupons []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		coupons = append(coupons, code)
	}

	resp := &couponpb.GetCampaignResponse{
		Campaign: &couponpb.Campaign{
			Id:           campaign.ID,
			Name:         campaign.Name,
			StartTime:    timestamppb.New(campaign.StartTime),
			TotalCoupons: campaign.TotalCoupons,
		},
		CouponCodes: coupons,
	}
	return connect.NewResponse(resp), nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

type Server struct {
	service *CouponService
	mux     *http.ServeMux
}

func New() (*Server, error) {
	dbHost := getEnv("DB_HOST", "127.0.0.1")
	dbPort := getEnv("DB_PORT", "3306")
	dbUser := getEnv("DB_USER", "coxwave")
	dbPassword := getEnv("DB_PASSWORD", "coxwavewave")
	dbName := getEnv("DB_NAME", "coupons")
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", 
		dbUser, dbPassword, dbHost, dbPort, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(50)
	db.SetConnMaxLifetime(5 * time.Minute)

	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort)
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		PoolSize: 100,
		MinIdleConns: 50,
	})

	service := NewCouponService(db, redisClient)
	s := &Server{
		service: service,
		mux:     http.NewServeMux(),
	}

	path, handler := couponpbconnect.NewCouponServiceHandler(service)
	s.mux.Handle(path, handler)

	return s, nil
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) Close() error {
	s.service.db.Close()
	s.service.redisClient.Close()
	return nil
}
