# Forum

A Go web forum built with SQLite.

## Current setup

The application opens the SQLite database at `internal/db/forum.db` during startup.
If the file does not exist, SQLite creates it. The application verifies the
connection before the HTTP server is started, so database startup errors appear
in the terminal instead of becoming browser errors.

## Prerequisites

- Go 1.26.2 or newer
- A C compiler (`gcc`)

The project uses `github.com/mattn/go-sqlite3`. This driver requires CGO and a
C compiler when Go builds the application.

On Ubuntu or Debian, install the compiler with:

```bash
sudo apt update
sudo apt install build-essential
```

## Run locally

From the repository root:

```bash
CGO_ENABLED=1 go run ./cmd
```

Expected startup output:

```text
Starting database...
Database connected successfully
Running migrations...
Migrations ran successfully
Server is running on http://localhost:8080
Press Ctrl + C to terminate the server...
```

## Run tests

From the repository root, run all tests with CGO enabled for SQLite:

```bash
CGO_ENABLED=1 go test ./...
```

The test suite uses temporary SQLite databases and covers database migrations,
HTTP route registration, authentication, posts, comments, reactions, and post
filters. It does not use the production database at `internal/db/forum.db`.

## Inspect the SQLite database

From the repository root, open the database shell:

```bash
sqlite3 internal/db/forum.db
```

Inside the SQLite prompt, these commands show the tables, a table definition,
and the stored rows:

```sql
.tables
.schema posts
.headers on
.mode column
SELECT * FROM users;
SELECT * FROM posts;
SELECT * FROM comments;
```

Exit the SQLite prompt with:

```sql
.quit
```

## Running on another computer

Commit and transfer `go.mod` and `go.sum` with the source code. On any machine
that builds the application, verify that `gcc` is installed, then run the same
`CGO_ENABLED=1 go run ./cmd` command.

## Run with Docker

From the repository root, build the image:

```bash
docker build -t forum .
```

Run the container:

```bash
docker run --name forum-container -p 8080:8080 forum
```

Open <http://localhost:8080>.
