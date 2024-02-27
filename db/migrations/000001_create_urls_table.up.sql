CREATE TABLE IF NOT EXISTS urls (
  id SERIAL PRIMARY KEY,
  project TEXT NOT NULL,
  url TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  finished_at TIMESTAMP,
  started_processing_at TIMESTAMP,
  worker TEXT,
  UNIQUE (project, url)
);
