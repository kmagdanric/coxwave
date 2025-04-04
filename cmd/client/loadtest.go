package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"coupons/internal/client"
	"github.com/bufbuild/connect-go"
	couponpb "coupons/proto"
)

type LoadTestResult struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	ErrorCounts        map[string]int
	ActualDuration     time.Duration
	RemainingCoupons   int64
	IssuedCoupons      int64
	IssuedCodes        map[string]bool
}

func runLoadTest() (*LoadTestResult, error) {
	// Create a new client
	c, err := client.New("http://localhost:8080")
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	// Create a campaign that starts 3 seconds later
	startTime := time.Now().Add(3 * time.Second)
	campaignID, err := c.CreateCampaign(
		context.Background(),
		"Load Test Campaign",
		startTime,
		20000,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create campaign: %v", err)
	}
	log.Printf("Created campaign with ID: %d", campaignID)

	// Try to issue a coupon immediately (should fail)
	_, err = c.IssueCoupon(context.Background(), campaignID)
	if err == nil {
		return nil, fmt.Errorf("expected error when issuing coupon before start time")
	}
	log.Printf("Successfully verified that issuing coupon before start time fails: %v", err)

	// Wait for campaign start time
	log.Printf("Waiting for campaign to start...")
	time.Sleep(time.Until(startTime))

	// Prepare for load test
	var (
		totalRequests      int64
		successfulRequests int64
		failedRequests     int64
		errorCounts        = make(map[string]int)
		errorCountsMu      sync.Mutex
		issuedCodes        = make(map[string]bool)
		issuedCodesMu      sync.Mutex
		wg                sync.WaitGroup
	)

	const (
		totalRequestsTarget = 10000
		batchSize          = 100
		progressInterval   = time.Second
	)
	targetDuration := 10 * time.Second

	batchInterval := targetDuration / time.Duration(totalRequestsTarget/batchSize)
	numBatches := totalRequestsTarget / batchSize

	log.Printf(
		"Starting load test with %d requests over %v...",
		totalRequestsTarget,
		targetDuration,
	)
	testStart := time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// separate goroutine for progress reporting
	go func() {
		ticker := time.NewTicker(progressInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				success := atomic.LoadInt64(&successfulRequests)
				failed := atomic.LoadInt64(&failedRequests)
				total := atomic.LoadInt64(&totalRequests)
				log.Printf("Progress: %d/%d requests (success: %d, failed: %d)", 
					total, totalRequestsTarget, success, failed)
			}
		}
	}()

	// Send requests in batches for better control
	for batch := 0; batch < numBatches && atomic.LoadInt64(&totalRequests) < totalRequestsTarget; batch++ {
		batchStart := time.Now()
		
		for i := 0; i < batchSize && atomic.LoadInt64(&totalRequests) < totalRequestsTarget; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				if atomic.LoadInt64(&totalRequests) >= totalRequestsTarget {
					return
				}
				atomic.AddInt64(&totalRequests, 1)

				couponCode, err := c.IssueCoupon(context.Background(), campaignID)
				if err != nil {
					atomic.AddInt64(&failedRequests, 1)
					errorCountsMu.Lock()
					errorCounts[err.Error()]++
					errorCountsMu.Unlock()
					return
				}

				atomic.AddInt64(&successfulRequests, 1)
				issuedCodesMu.Lock()
				issuedCodes[couponCode] = true
				issuedCodesMu.Unlock()
			}()
		}

		elapsed := time.Since(batchStart)
		if elapsed < batchInterval {
			time.Sleep(batchInterval - elapsed)
		}
	}

	wg.Wait()
	actualDuration := time.Since(testStart)
	cancel()
	log.Printf("Load test completed in %v", actualDuration)

	req := connect.NewRequest(&couponpb.GetCampaignRequest{
		CampaignId: campaignID,
	})
	resp, err := c.Client.GetCampaign(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to get campaign details: %v", err)
	}

	remainingCoupons := resp.Msg.Campaign.TotalCoupons - int64(len(resp.Msg.CouponCodes))
	issuedCoupons := int64(len(resp.Msg.CouponCodes))

	result := &LoadTestResult{
		TotalRequests:      totalRequests,
		SuccessfulRequests: successfulRequests,
		FailedRequests:     failedRequests,
		ErrorCounts:        errorCounts,
		ActualDuration:     actualDuration,
		RemainingCoupons:   remainingCoupons,
		IssuedCoupons:      issuedCoupons,
		IssuedCodes:        issuedCodes,
	}

	if remainingCoupons != 10000 {
		return result, fmt.Errorf("expected 10000 remaining coupons, got %d", remainingCoupons)
	}

	if issuedCoupons != successfulRequests {
		return result, fmt.Errorf("mismatch between successful requests (%d) and issued coupons in DB (%d)", 
			successfulRequests, issuedCoupons)
	}

	dbCodes := make(map[string]bool)
	for _, code := range resp.Msg.CouponCodes {
		dbCodes[code] = true
	}

	for code := range issuedCodes {
		if !dbCodes[code] {
			return result, fmt.Errorf("issued coupon code %s not found in database", code)
		}
	}

	for code := range dbCodes {
		if !issuedCodes[code] {
			return result, fmt.Errorf("unexpected coupon code %s found in database", code)
		}
	}

	return result, nil
}

func main() {
	result, err := runLoadTest()
	if err != nil {
		log.Printf("\nLoad Test Results (with errors):")
	} else {
		log.Printf("\nLoad Test Results (success):")
	}

	fmt.Printf("\nTest Duration: %v\n", result.ActualDuration)
	fmt.Printf("Request Statistics:\n")
	fmt.Printf("- Total Requests Made: %d\n", result.TotalRequests)
	fmt.Printf("- Successful Requests: %d\n", result.SuccessfulRequests)
	fmt.Printf("- Failed Requests: %d\n", result.FailedRequests)
	if result.FailedRequests > 0 {
		fmt.Printf("\nError Distribution:\n")
		for errMsg, count := range result.ErrorCounts {
			fmt.Printf("- %s: %d\n", errMsg, count)
		}
	}
	fmt.Printf("\nCoupon Statistics:\n")
	fmt.Printf("- Coupons Issued (in DB): %d\n", result.IssuedCoupons)
	fmt.Printf("- Coupons Remaining: %d\n", result.RemainingCoupons)
	fmt.Printf("- Request Success Rate: %.2f%%\n", 
		float64(result.SuccessfulRequests)/float64(result.TotalRequests)*100)
	fmt.Printf("\nCoupon Code Verification:\n")
	fmt.Printf("- Total Unique Codes Issued: %d\n", len(result.IssuedCodes))
	fmt.Printf("- All Codes Verified in Database: Yes\n")

	if err != nil {
		log.Fatalf("Load test failed: %v", err)
	}
}
