import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    constant_load: {
      executor: 'constant-vus',
      vus: 100,
      duration: '10s',
      gracefulStop: '2s',
    },
  },
  thresholds: {
    'http_req_duration': ['p(95)<500'],
  },
  noConnectionReuse: true,
  batch: 20,
  batchPerHost: 20,
};

// Helper function to make Connect requests
function connectRequest(service, method, message) {
  const url = `http://localhost:8080/${service}/${method}`;
  const headers = {
    'Content-Type': 'application/json',
    'Connect-Protocol-Version': '1',
    'Connect-Timeout-Ms': '10000',
  };
  
  return http.post(url, JSON.stringify(message), { headers });
}

// Setup function - runs once at the beginning
export function setup() {
  // Create a campaign that starts in 3 seconds
  const startTime = new Date(Date.now() + 3000);
  
  const createResp = connectRequest('couponpb.CouponService', 'CreateCampaign', {
    name: "K6 Load Test Campaign",
    startTime: startTime.toISOString(),
    totalCoupons: 20000,
  });

  check(createResp, {
    'campaign created successfully': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.campaignId > 0;
      } catch (e) {
        return false;
      }
    },
  });

  let campaignId;
  try {
    const body = JSON.parse(createResp.body);
    campaignId = body.campaignId;
  } catch (e) {
    throw new Error(`Failed to parse campaign ID from response: ${createResp.body}`);
  }

  // Verify that issuing before start time fails
  const earlyIssueResp = connectRequest('couponpb.CouponService', 'IssueCoupon', {
    campaignId: campaignId,
  });

  check(earlyIssueResp, {
    'early issuance properly rejected': (r) => {
      try {
        const body = JSON.parse(r.body);
        return r.status === 500 && (
          body.message === 'coupon issuance not started yet' ||
          body.message.includes('no more coupons available')
        );
      } catch (e) {
        return false;
      }
    },
  });

  // Wait for campaign start time
  sleep(3);
  
  return { campaignId };
}

// Default function - runs for each virtual user
export default function (data) {
  const resp = connectRequest('couponpb.CouponService', 'IssueCoupon', {
    campaignId: data.campaignId,
  });

  check(resp, {
    'request handled properly': (r) => {
      try {
        if (r.status === 200) {
          const body = JSON.parse(r.body);
          return body.couponCode && body.couponCode.length > 0;
        } else if (r.status === 500) {
          const body = JSON.parse(r.body);
          return body.message && (
            body.message === 'no more coupons available' || 
            body.message === 'coupon issuance not started yet'
          );
        }
        return false;
      } catch (e) {
        console.log(`Error parsing response: ${e.message}, body: ${r.body}`);
        return false;
      }
    },
  });
}

// Teardown function - runs once at the end
export function teardown(data) {
  const campaignResp = connectRequest('couponpb.CouponService', 'GetCampaign', {
    campaignId: data.campaignId,
  });

  check(campaignResp, {
    'campaign data retrieved': (r) => {
      try {
        const body = JSON.parse(r.body);
        if (!body.campaign) {
          console.log('Response is missing campaign field');
          return false;
        }
        if (body.campaign.id !== data.campaignId) {
          console.log(`Campaign ID mismatch: expected ${data.campaignId}, got ${body.campaign.id}`);
          return false;
        }
        return true;
      } catch (e) {
        console.error('Failed to parse GetCampaign response:', e);
        return false;
      }
    },
    'correct number of coupons issued': (r) => {
      try {
        const body = JSON.parse(r.body);
        const issuedCount = body.couponCodes.length || 0;
        const totalCoupons = body.campaign.totalCoupons;
        const remainingCoupons = totalCoupons - issuedCount;

        console.log(`Campaign Statistics:`);
        console.log(`- Total Coupons Available: ${totalCoupons}`);
        console.log(`- Coupons Issued: ${issuedCount}`);
        console.log(`- Coupons Remaining: ${remainingCoupons}`);

        return remainingCoupons == 0;
      } catch (e) {
        console.log(e)
        return false;
      }
    },
  });
}
