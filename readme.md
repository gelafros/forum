# Forum

A Go web forum built with SQLite.

## Current setup

The application opens the SQLite database at `db/forum.db` during startup.
If the file does not exist, SQLite creates it. The application verifies the
connection before the HTTP server is started, so database startup errors appear
in the terminal instead of becoming browser errors.

## Prerequisites

- Go 1.26 or compatible
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
CGO_ENABLED=1 go run .
```

Expected startup output:

```text
Forum Database connected successfully
```

## Inspect the SQLite database

From the repository root, open the database shell:

```bash
sqlite3 db/forum.db
```

Inside the SQLite prompt, these commands show the tables, a table definition,
and the stored rows:

```sql
.tables
.schema posts
.headers on
.mode column
SELECT * FROM schema_migrations;
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
`CGO_ENABLED=1 go run .` command. Docker will need a C compiler in its build
environment too.
