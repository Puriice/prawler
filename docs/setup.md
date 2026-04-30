[back](../README.md)

# Guide to setup crawler

## Prerequisites

| Service    | Where | Notes                  |
| ---------- | ----- | ---------------------- |
| PostgreSQL | any   |                        |
| RabbitMQ   | any   | default guest/guest    |
| Go 1.25.6+ | any   | runs/build the crawler |

## Configuration

Runtime settings live in `config.json`. The secrets configuration are read from the environment variable and must never appear in `config.json`.

```json
{
    "version": "1.3",
    "contact": {
        "email": "purinutt.amartayavis@g.swu.ac.th",
        "description": "Educational Project",
        "git": "https://github.com/Puriice/prawler"
    },
    "exchange_name": {
        "uri": "prawler.uri",
        "blacklists": "prawler.blacklists",
        "frontier": "prawler.frontier",
        "backoff": "prawler.backoff",
        "embedding": "prawler.embedding"
    },
    "queue_name": {
        "uri": "prawler.uri",
        "backoff": "prawler.backoff"
    },
    "policy": {
        "user_agent": "Prawler",
        "backoff": {
            "jitter": 0.5,
            "multiplier_429": 2,
            "multiplier_503": 3,
            "multiplier_5XX": 2.5
        },
        "minimum_crawling_delay_ms": 3000,
        "maximum_crawling_delay_ms": 60000,
        "maximum_crawling_depth": 5,
        "maximum_crawling_attempt": 5
    }
}
```

## Environment Variables

### Shared on both nodes

```env
DB_URL=<POSTGRES CONN STRING>
AMQP_URL=<RABBITMQ CONN STRING DEFAULT "amqp://guest:guest@localhost/">
```

### Only Crawling Master

```env
HOST=<HTTP HOST>
PORT=<HTTP PORT DEFAULT "5723">
```

### Only Crawling Slave

```env
HOLTER_URL=<MASTER NODE URL DEFAULT "http://localhost:5723">
```

## Blacklists

Blacklist all domain and sub-domain in the lists. Preventing crawler from crawling restricted website.

```json
["http://example.com"]
```

## Setup

### Downloading dependency using Golang

```bash
go mod download
```

## Crawler Master Node

This node is acting as a url frontier; filter url, deduplicate, and distributed works between crawlers slaves. **The master must be a single process**

```bash
# run master with default config path ./config.json and blacklist ./blacklists.json
go run ./cmd/crawler/master

# run master with custom configuration path
go run ./cmd/crawler/master --config ./configuration.json

# run master with custom blacklists path
go run ./cmd/crawler/master --blacklist ./blacklists.json

# run master with custom configuration and blacklists path
go run ./cmd/crawler/master --config ./config.json --blacklist ./blacklist.json
```

## Crawler Slave Node

This node is responsible for fetching, parsing, chunking the page contents, and storing the parsing data to the database.
The crawler can have multiple nodes reporting to the single master node.

```bash
# run slave with default config pointed to ./config.json
go run ./cmd/crawler/slave

# run slave with custom configuration path
go run ./cmd/crawler/slave --config ./configuration.json
```

## Producer

The tool for register the initial seeds for crawling master.

```bash
# run producer with default to ./seeds.json
go run ./cmd/producer

# run producer with custom seeds path
go run ./cmd/producer --seed-file ./cool-seeds.json
```

## Producer/Embedder

The tool for queuing unembedded/old embedded version to the embedding queue. Using only in migration or lose queue data.

```bash
go run ./cmd/producer/embedding
```
