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
