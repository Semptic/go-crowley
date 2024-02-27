package main

import (
  "context"
	"log"
  "flag"
  "fmt"
  "os"


	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

  "github.com/jackc/pgx/v5"
)

func main() {
  dbUrl := "postgres://postgres:example@localhost:5432/postgres?sslmode=disable"

  m, err := migrate.New("file://db/migrations", dbUrl)
	if err != nil {
		log.Fatal(err)
	}

  if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal(err)
  }

  var projectFlag = flag.String("p", "", "Project ID")
  var urlFlag = flag.String("u", "", "Start URL")

  flag.Parse()

  if *projectFlag == "" {
    log.Fatal("Project ID is required")
    os.Exit(1)
  }
  if *urlFlag == "" {
    log.Fatal("Start URL is required")
    os.Exit(1)
  }

  fmt.Println("Project ID:", *projectFlag)
  fmt.Println("Start URL:", *urlFlag)

  conn, err := pgx.Connect(context.Background(), dbUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

  args := pgx.NamedArgs{"project": *projectFlag, "url": *urlFlag}

	_, err = conn.Exec(context.Background(), "INSERT INTO urls (project, url) VALUES (@project, @url)", args)

	if err != nil {
		fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
		os.Exit(1)
	}
}
