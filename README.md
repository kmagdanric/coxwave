## Coxwave Coupons System

This is a coupon issuing system capable of handling more than 1000 requests per second.
System is created as a part of Coxwave Programming Assignment.

## Installation

This project used following softwares and assumes they are installed:
- docker (version 27.3.1)
- go (version 1.24.2)

This project uses:
- buf (version 1.28.1)

You can install buf and go plugins using script:
```{sh}
./install.sh
```

## Quickstart

You can launch the local server environment using docker-compose.
```{sh}
./docker-compose.py build && ./docker-compose.py up -d
```

You can then run go client code to run loadtest. After loadtest is conducted, summarized result is available in stdout.
```{sh}
go mod tidy
go run cmd/client/loadtest.go
```

## Initial Software Design

I thought of creating a connectrpc go server, RDBMS, and some in-memory cache for atomic counter update.

I have assumed the following:
- start of the coupon campaign is where the traffic is the highest.

One major problem initially was keeping track of number of coupons issued for a given campaign, so that no excess coupons are issued. We had two strategies:

1) pre-creating all the coupons when creating a campaign

It would slow down database when updating coupon as "active". Multiple requests at once would have to be processed one by one, as database has to get a table-granularity read lock for all the coupons that are not active. Consequently, db updates would be a bottleneck.

2) creating a coupon when IssueCoupon request comes in

It has another issue: just letting RDBMS search for number of coupons in a campaign every time IssueCoupon request comes in is very slow. I didn't think go server would be able to handle 1000 requests per second.

Redis is inherently single thread in its main event loop and has atomic INCR and DECR operations that is exactly what I was looking for.

## Project Structure

Project has following structure:

```
cmd/
    client/
        main.go
        loadtest.go // loadtest script
    server/
        main.go // connectrpc server
internal/
    client/
        main.go
    server/
        main.go
proto/ // contains coupons.proto for connectrpc/proto generation
mysql/ // contains database user/schema loaded as docker volume (not persisted across session)
docker/ // contains docker-compose.yaml used in docker-compose.py
generate-proto.sh // generates *pb.go files for local development
install.sh // installs dependencies for the project
docker-compose.py // runs docker compose for testing
buf.gen.yaml
buf.yaml // buf related configs
go.mod
go.sum
```

## Loadtest Thoughts

Loadtest was conducted using golang client - connectrpc supports browser, node.js, and browser at the same time. I wanted to emulated simultaneous requests, not necessarily involving the rotating userbase. Therefore, I chose to use goroutine for sending protobuf requests, not JSON.

## Hash Collision

Initial approach generated coupon codes with following format:
KKNNNNKKNN (K: Korean, N: Number)

Korean characters were chosen randomly from a Korean Character list, while numbers were as well. The randomness came from Sha256 hash of (CampaignID, Request Timestamp). Problems were expected as random number is seeded, but in this case seeding the hash seemed better because request timestamp is nanoseconds and I thought it would be okay to try.

```
# coxwave git:(main) ✗ go run cmd/client/loadtest.go
2025/04/04 16:36:19 Created campaign with ID: 5
2025/04/04 16:36:19 Successfully verified that issuing coupon before start time fails: unknown: coupon issuance not started yet
2025/04/04 16:36:19 Waiting for campaign to start...
2025/04/04 16:36:22 Starting load test with 10000 requests over 10s...
2025/04/04 16:36:23 Progress: 1000/10000 requests (success: 999, failed: 1)
2025/04/04 16:36:24 Progress: 2000/10000 requests (success: 1999, failed: 1)
2025/04/04 16:36:25 Progress: 3000/10000 requests (success: 2999, failed: 1)
2025/04/04 16:36:26 Progress: 4000/10000 requests (success: 3998, failed: 2)
2025/04/04 16:36:27 Progress: 5000/10000 requests (success: 4998, failed: 2)
2025/04/04 16:36:28 Progress: 6000/10000 requests (success: 5998, failed: 2)
2025/04/04 16:36:29 Progress: 7000/10000 requests (success: 6997, failed: 3)
2025/04/04 16:36:30 Progress: 8000/10000 requests (success: 7996, failed: 4)
2025/04/04 16:36:31 Progress: 9000/10000 requests (success: 8995, failed: 5)
2025/04/04 16:36:32 Progress: 10000/10000 requests (success: 9994, failed: 5)
2025/04/04 16:36:32 Load test completed in 10.082427792s
2025/04/04 16:36:32 
Load Test Results (with errors):

Test Duration: 10.082427792s
Request Statistics:
- Total Requests Made: 10000
- Successful Requests: 9995
- Failed Requests: 5

Error Distribution:
- unknown: failed to persist coupon issuance: Error 1062 (23000): Duplicate entry '내재1657도소53' for key 'coupons.idx_coupon_code': 1
- unknown: failed to persist coupon issuance: Error 1062 (23000): Duplicate entry '캐새6720오모48' for key 'coupons.idx_coupon_code': 1
- unknown: failed to persist coupon issuance: Error 1062 (23000): Duplicate entry '모나7659노마69' for key 'coupons.idx_coupon_code': 1
- unknown: failed to persist coupon issuance: Error 1062 (23000): Duplicate entry '타카4523채조49' for key 'coupons.idx_coupon_code': 1
- unknown: failed to persist coupon issuance: Error 1062 (23000): Duplicate entry '보노4331다포80' for key 'coupons.idx_coupon_code': 1

Coupon Statistics:
- Coupons Issued (in DB): 9995
- Coupons Remaining: 10005
- Request Success Rate: 99.95%
2025/04/04 16:36:32 Load test failed: expected 10000 remaining coupons, got 10005
exit status 1
```

It was not okay to make timestamp as hash seeds, so I added 8 random bytes.
After applying little more entropy, I could successfully make every coupon unique.

```
# coxwave git:(main) ✗ go run cmd/client/loadtest.go
2025/04/04 16:40:58 Created campaign with ID: 1
2025/04/04 16:40:58 Successfully verified that issuing coupon before start time fails: unknown: coupon issuance not started yet
2025/04/04 16:40:58 Waiting for campaign to start...
2025/04/04 16:41:01 Starting load test with 10000 requests over 10s...
2025/04/04 16:41:02 Progress: 1000/10000 requests (success: 1000, failed: 0)
2025/04/04 16:41:03 Progress: 2000/10000 requests (success: 2000, failed: 0)
2025/04/04 16:41:04 Progress: 3000/10000 requests (success: 3000, failed: 0)
2025/04/04 16:41:05 Progress: 4000/10000 requests (success: 4000, failed: 0)
2025/04/04 16:41:06 Progress: 5000/10000 requests (success: 5000, failed: 0)
2025/04/04 16:41:07 Progress: 6000/10000 requests (success: 6000, failed: 0)
2025/04/04 16:41:08 Progress: 7000/10000 requests (success: 7000, failed: 0)
2025/04/04 16:41:09 Progress: 8000/10000 requests (success: 8000, failed: 0)
2025/04/04 16:41:10 Progress: 9000/10000 requests (success: 9000, failed: 0)
2025/04/04 16:41:11 Progress: 10000/10000 requests (success: 10000, failed: 0)
2025/04/04 16:41:11 Load test completed in 10.086413084s
2025/04/04 16:41:11 
Load Test Results (success):

Test Duration: 10.086413084s
Request Statistics:
- Total Requests Made: 10000
- Successful Requests: 10000
- Failed Requests: 0

Coupon Statistics:
- Coupons Issued (in DB): 10000
```

Overall, loadtest proved to be successful with ~1000 requests per second. Exactly 10000 coupons were issued and there was no data integrity issue as GetCampaign result and database content matched. No requests have failed because of DB connection pool running out or Golang server could not process new request.

## Improvements

By no means this loadtest is adequate: several critial areas could be tested further:

1. Use K6 for testing
- supports various load patterns
- built in metrics collection/visualization
- Javascript based test scripting

2. Test scenarios
1) Multi-campaign concurrent load
2) Variable load patterns (traffic burst, ramp-up, etc)
3) Edge cases
- campaign start time boundary conditions
- near-limit coupon issuance scenarios
