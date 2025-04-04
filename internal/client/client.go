package client

import (
	"context"
	"net/http"
	"time"

	"github.com/bufbuild/connect-go"
	"google.golang.org/protobuf/types/known/timestamppb"

	couponpb "coupons/proto"
	"coupons/proto/couponpbconnect"
)

// Client wraps the coupon service client
type Client struct {
	Client couponpbconnect.CouponServiceClient
}

// New creates a new Client instance
func New(baseURL string) (*Client, error) {
	return &Client{
		Client: couponpbconnect.NewCouponServiceClient(
			http.DefaultClient,
			baseURL,
		),
	}, nil
}

// CreateCampaign creates a new campaign
func (c *Client) CreateCampaign(ctx context.Context, name string, startTime time.Time, totalCoupons int64) (int64, error) {
	req := connect.NewRequest(&couponpb.CreateCampaignRequest{
		Name:         name,
		StartTime:    timestamppb.New(startTime),
		TotalCoupons: totalCoupons,
	})

	resp, err := c.Client.CreateCampaign(ctx, req)
	if err != nil {
		return 0, err
	}
	return resp.Msg.CampaignId, nil
}

// GetCampaign gets campaign details
func (c *Client) GetCampaign(ctx context.Context, campaignID int64) (*couponpb.Campaign, error) {
	req := connect.NewRequest(&couponpb.GetCampaignRequest{
		CampaignId: campaignID,
	})

	resp, err := c.Client.GetCampaign(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Msg.Campaign, nil
}

// IssueCoupon issues a new coupon
func (c *Client) IssueCoupon(ctx context.Context, campaignID int64) (string, error) {
	req := connect.NewRequest(&couponpb.IssueCouponRequest{
		CampaignId: campaignID,
	})

	resp, err := c.Client.IssueCoupon(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Msg.CouponCode, nil
} 