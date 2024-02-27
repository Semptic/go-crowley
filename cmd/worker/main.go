package main

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/net/html"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (state *State) crawl(foundUrls chan UrlData, urlsToCrawl chan UrlData) {
	for url := range urlsToCrawl {
		state.crawlPage(foundUrls, url)
	}
}

func (state *State) crawlPage(foundURLs chan UrlData, urlData UrlData) error {
	fmt.Println("Crawling URL:", urlData)

	res, err := http.Get(urlData.url.String())

	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to fetch URL: %v\n", err)
		return err
	}
	defer res.Body.Close()

	fmt.Println("Fetched URL:", urlData)

	z := html.NewTokenizer(res.Body)

	for {
		// If the HTML has ended, we break out of the loop
		token := z.Next()
		if token == html.ErrorToken {
			break
		}

		// New Token started
		if token == html.StartTagToken {
			// Check if the token is an <a> tag
			if name, _ := z.TagName(); string(name) == "a" {
				for {
					// Get the next attribute
					name, val, more := z.TagAttr()

					// Check if the attribute is "href"
					if string(name) == "href" {
						// Cast Url
						u, err := url.Parse(string(val))

						if err != nil {
							fmt.Fprintf(os.Stderr, "Unable to parse URL: %v\n", err)
							continue
						}

						resolved := urlData.url.ResolveReference(u)

						if resolved.Hostname() != urlData.url.Hostname() {
							continue
						}

						foundURLs <- UrlData{url: *resolved, project: urlData.project}
					}

					// There are no more attributes so we break out of the
					// attribute search loop.
					if !more {
						break
					}
				}
			}
		}
	}

	state.markUrlAsCrawled(urlData)

	return nil
}

func (state *State) dedublicateAndStoreUrls(newUrls chan UrlData) {
	for url := range newUrls {
		args := pgx.NamedArgs{"project": url.project, "url": url.url.String()}

		// fmt.Println("Inserting URL:", url)
		_, err := state.pool.Exec(context.Background(), "INSERT INTO urls (project, url) VALUES (@project, @url)", args)

		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				if pgErr.Code != "23505" {
					fmt.Fprintf(os.Stderr, "Inserting failed: %v\n", err)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Inserting failed: %v\n", err)
			}
		}
	}
}

func (state *State) markUrlAsCrawled(urlData UrlData) {
	fmt.Println("Marking URL as crawled:", urlData)

	query := "UPDATE urls SET finished_at = NOW() WHERE project = @project AND url = @url"

	args := pgx.NamedArgs{"project": urlData.project, "url": urlData.url.String()}  

	_, err := state.pool.Exec(context.Background(), query, args)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Marking failed: %v\n", err)
	}
}

func (state *State) resetTimedOutUrls() {
	for {
		query := `
      UPDATE urls SET started_processing_at = NULL
      WHERE finished_at IS NULL AND started_processing_at < NOW() - INTERVAL '5 minutes'
    `
		_, err := state.pool.Exec(context.Background(), query)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Resetting failed: %v\n", err)
		}

    time.Sleep(5 * time.Minute)
	}
}

type UrlData struct {
  url url.URL
  project string
}

func (state *State) loadNewUrls(urlsToCrawl chan UrlData) {
	query := `
      UPDATE urls SET started_processing_at = NOW()
      WHERE urls.id IN (
        SELECT id
        FROM urls
        WHERE
          finished_at IS NULL AND 
          started_processing_at IS NULL AND
          project IN (
            SELECT project
            FROM urls
            WHERE
              finished_at IS NULL
            GROUP BY project
            HAVING COUNT(started_processing_at) < 2
          )
        ORDER BY created_at 
        FOR UPDATE SKIP LOCKED
        LIMIT 1
      )
      RETURNING url, project;
    `
	for {
		rows, err := state.pool.Query(context.Background(), query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to query: %v\n", err)
		}
		defer rows.Close()

    gotRows := false

		for rows.Next() {
      var page string
      var project string

			err := rows.Scan(&page, &project)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to scan row: %v\n", err)
			}

      pageUrl, err := url.Parse(page)

      if err != nil {
        fmt.Fprintf(os.Stderr, "Unable to parse URL: %v\n", err)

        continue
      }

      data := UrlData{url: *pageUrl, project: project}

			fmt.Println("Got URL:", data)

			urlsToCrawl <- data 
      gotRows = true
		}

    if (!gotRows) {
		  time.Sleep(1 * time.Second)
    }
	}
}

type State struct {
	pool    *pgxpool.Pool
}

func main() {
	dbUrl := "postgres://postgres:example@localhost:5432/postgres?sslmode=disable"

	pool, err := pgxpool.New(context.Background(), dbUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	state := State{pool: pool}

	urlsToCrawl := make(chan UrlData, 12)
	foundUrls := make(chan UrlData, 32)

	go state.dedublicateAndStoreUrls(foundUrls)
	go state.crawl(foundUrls, urlsToCrawl)
	go state.loadNewUrls(urlsToCrawl)
  go state.resetTimedOutUrls()

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel
	//time for cleanup before exit
	fmt.Println("Adios!")
}
