package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/a-h/templ"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/Semptic/crowley/cmd/monitor/templates"
)

type State struct {
	pool *pgxpool.Pool
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
	fmt.Println("Hello, World!")

	app := echo.New()

	app.Use(middleware.Logger())
	app.Use(middleware.Recover())

	app.GET("/", state.MonitorHandler)
	app.GET("/overview", state.OverviewHandler)
	app.GET("/urls_in_progress", state.UrlsInProgressHandler)
	app.GET("/urls_in_queue", state.UrlsInQueueHandler)
	app.GET("/urls_finished", state.UrlsFinishedHandler)

	app.Logger.Fatal(app.Start(":4000"))
}

func (state *State) getOverview() (view.OverviewInput, error) {
	query := `
    SELECT
      COUNT(url) FILTER (WHERE finished_at IS NOT NULL) AS urls_finished,
      COUNT(url) FILTER (WHERE finished_at IS NULL AND started_processing_at IS NOT NULL) AS urls_in_progress,
      COUNT(url) FILTER (WHERE finished_at IS NULL AND started_processing_at IS NULL) AS urls_in_queue,
      COUNT(DISTINCT project) FILTER (WHERE finished_at IS NOT NULL) AS projects_finished,
      COUNT(DISTINCT project) FILTER (WHERE finished_at IS NULL AND started_processing_at IS NOT NULL) AS projects_in_progress
    FROM urls
  `

	overview := view.OverviewInput{}

	err := state.pool.QueryRow(context.Background(), query).Scan(
		&overview.UrlsCompleted, &overview.UrlsInProgress, &overview.UrlsQueued,
		&overview.ProjectsCompleted, &overview.ProjectsInProgress,
	)

	if err != nil {
		return overview, err
	}

	return overview, nil
}

func (state *State) getUrls(query string) ([]view.UrlData, error) {
  var urls []view.UrlData

	rows, err := state.pool.Query(context.Background(), query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to query: %v\n", err)
    return urls, err
	}
	defer rows.Close()


	for rows.Next() {
		var page string
		var project string

		err := rows.Scan(&page, &project)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to scan row: %v\n", err)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to parse URL: %v\n", err)

			continue
		}

		data := view.UrlData{Url: page, Project: project}

    urls = append(urls, data)
	}

  return urls, nil
}

func (state *State) getUrlsInProgress() ([]view.UrlData, error) {
	query := `
    SELECT url, project
    FROM urls
    WHERE
      finished_at IS NULL AND started_processing_at IS NOT NULL
    ORDER BY started_processing_at ASC
    LIMIT 10
  `

  return state.getUrls(query)
}

func (state *State) getUrlsInQueue() ([]view.UrlData, error) {
	query := `
    SELECT url, project
    FROM urls
    WHERE
      finished_at IS NULL AND started_processing_at IS NULL
    ORDER BY created_at DESC
    LIMIT 10
  `

  return state.getUrls(query)
}

func (state *State) getUrlsFinished() ([]view.UrlData, error) {
	query := `
    SELECT url, project
    FROM urls
    WHERE
      finished_at IS NOT NULL
    ORDER BY finished_at DESC
    LIMIT 10
  `

  return state.getUrls(query)
}

func (state *State) OverviewHandler(c echo.Context) error {
  overview, err := state.getOverview()

	if err != nil {
		return err
	}

	return Render(c, http.StatusOK, view.Overview(overview))
}

func (state *State) UrlsInProgressHandler(c echo.Context) error {
  urls, err := state.getUrlsInProgress()

	if err != nil {
		return err
	}

	return Render(c, http.StatusOK, view.Urls(urls))
}

func (state *State) UrlsInQueueHandler(c echo.Context) error {
  urls, err := state.getUrlsInQueue()

	if err != nil {
		return err
	}

	return Render(c, http.StatusOK, view.Urls(urls))
}

func (state *State) UrlsFinishedHandler(c echo.Context) error {
  urls, err := state.getUrlsFinished()

	if err != nil {
		return err
	}

	return Render(c, http.StatusOK, view.Urls(urls))
}

func (state *State) MonitorHandler(c echo.Context) error {
	return Render(c, http.StatusOK, view.MonitorPage())
}

func Render(ctx echo.Context, statusCode int, t templ.Component) error {
	ctx.Response().Writer.WriteHeader(statusCode)
	ctx.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	return t.Render(ctx.Request().Context(), ctx.Response().Writer)
}
