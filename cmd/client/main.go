package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"coupons/internal/client"
)

func main() {
	// Create a new client
	c, err := client.New("http://localhost:8080")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Create a campaign
	startTime := time.Now().Add(3 * time.Second)
	campaignID, err := c.CreateCampaign(context.Background(), "Test Campaign", startTime, 1000)
	if err != nil {
		log.Fatalf("Failed to create campaign: %v", err)
	}
	fmt.Printf("Created campaign with ID: %d\n", campaignID)

	// Get campaign details
	campaign, err := c.GetCampaign(context.Background(), campaignID)
	if err != nil {
		log.Fatalf("Failed to get campaign: %v", err)
	}
	fmt.Printf("Campaign details: %+v\n", campaign)

	// Wait for campaign start time
	fmt.Println("Waiting for campaign to start...")
	time.Sleep(time.Until(startTime.Add(3 * time.Second)))

	// Issue a coupon
	couponCode, err := c.IssueCoupon(context.Background(), campaignID)
	if err != nil {
		log.Fatalf("Failed to issue coupon: %v", err)
	}
	fmt.Printf("Issued coupon with code: %s\n", couponCode)
} 