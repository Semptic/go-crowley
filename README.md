# Crowley

## Run

1. Start postgres
```
docker compose up -d
```

2. Add couple of domains to crawl
```sh
go run cmd/manager/*.go -p wiki -u https://en.wikipedia.org/wiki/Music_genre
go run cmd/manager/*.go -p hackernews -u https://news.ycombinator.com/
```

3. Start the monitor
```sh
./watch_monitor.sh
```
or
```sh
go run cmd/monitor/*.go
```

4. Start some worker (run the following in multiple terminals)
```sh
go run cmd/worker/*.go
```

## Setup


### Prerequisites

Install go, templ, [air](https://github.com/cosmtrek/air?tab=readme-ov-file#installation) (Until [this](https://github.com/cosmtrek/air/pull/512) is merged install `git clone -b feat-live-proxy https://github.com/ndajr/air && cd air && go install .`).


### DB

Use [migrate](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate) for handling database migrations 
and [pgx](https://github.com/jackc/pgx/tree/master) for postgres.

```
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

#### Create new migration file

```
migrate create -dir db/migrations -ext sql [NAME]
```

#### Drop database

```
USER=<user>; PASS=<pass> migrate -database "postgres://$USER:$PASS@localhost:5432/galaxy?sslmode=disable" drop
```

