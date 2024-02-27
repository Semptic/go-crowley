# galactic_pioneers

## Setup


### Prerequisites

Install go, templ, [air](https://github.com/cosmtrek/air?tab=readme-ov-file#installation) (Until [this](https://github.com/cosmtrek/air/pull/512) is merged install `git clone -b feat-live-proxy https://github.com/ndajr/air && cd air && go install .`)
and [tailwindcss](https://tailwindcss.com/docs/installation) (`npx tailwindcss` is just fine).


### DB

Use [migrate](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate) for handling database migrations 
and [jet](https://github.com/go-jet/jet) for typed queries.

```
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

#### Create new migration file

```
migrate create -dir migrations -ext sql [NAME]
```


#### Apply migrations

```
USER=<user>; PASS=<pass> migrate -source file://migrations -database "postgres://$USER:$PASS@localhost:5432/galaxy?sslmode=disable" up
```

#### Drop database

```
USER=<user>; PASS=<pass> migrate -database "postgres://$USER:$PASS@localhost:5432/galaxy?sslmode=disable" drop
```

