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
Database connected successfully
```

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

Install Docker and start its daemon. Run these commands from the repository
root. Go and a C compiler are installed inside the builder image, so they are
not required on the host for the Docker workflow.

Build the image:

```bash
docker build -t forum .
```

Create and start the container in the background:

```bash
docker run -d --name forum-app -p 127.0.0.1:8080:8080 \
  --mount type=volume,source=forum-data,target=/app/internal/db forum
```

Open <http://localhost:8080>. If port 8080 is already in use, choose another host
port, for example `127.0.0.1:8081:8080`, and open <http://localhost:8081>.

Docker creates the named volume `forum-data` automatically. On its first use,
the image's migrations are copied into the volume, and the application creates
`forum.db` there. This database is separate from your local development database.
The volume retains data when the container is removed. See the
[Docker volume documentation](https://docs.docker.com/engine/storage/volumes/).

Check the container and its startup logs:

```bash
docker ps
docker logs -f forum-app
```

Press Ctrl+C to stop following the logs. The container keeps running.

Stop and restart the existing container:

```bash
docker stop forum-app
docker start forum-app
```

After changing the code, rebuild the image and replace the container:

```bash
docker stop forum-app
docker rm forum-app
docker build -t forum .
docker run -d --name forum-app -p 127.0.0.1:8080:8080 \
  --mount type=volume,source=forum-data,target=/app/internal/db forum
```

Reusing `forum-data` preserves the database. Because the volume also contains
the migrations directory, existing volumes retain their original SQL files;
changes to migrations require updating those files in the volume separately.
