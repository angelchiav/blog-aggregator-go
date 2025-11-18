# Blog Aggregator (gator)

A command-line RSS feed aggregator built with Go. This tool allows you to register users, add RSS feeds, follow feeds, and browse aggregated posts from your followed feeds.

## Prerequisites

Before you can run this program, you'll need:

- **PostgreSQL**: A running PostgreSQL database instance
- **Go**: Go 1.25.1 or later installed on your system

## Installation

Install the `gator` CLI tool using `go install`:

```bash
go install github.com/angelchiav/blog-aggregator-go@latest
```

After installation, the `gator` binary will be available in your `$GOPATH/bin` directory (or `$HOME/go/bin` by default). Make sure this directory is in your `$PATH` to run `gator` from anywhere.

## Configuration

Before running the program, you need to set up a configuration file. The program looks for a config file at `~/.gatorconfig.json` (or you can set the `GATOR_CONFIG` environment variable to specify a different path).

Create the config file with the following structure:

```json
{
 "db_url": "postgres://username:password@localhost:5432/dbname?sslmode=disable",
 "current_user_name": ""
}
```

Replace the `db_url` with your actual PostgreSQL connection string. The `current_user_name` will be set automatically when you log in.

### Example config file:

```json
{
 "db_url": "postgres://postgres:password@localhost:5432/blog_aggregator?sslmode=disable",
 "current_user_name": ""
}
```

## Database Setup

Make sure your PostgreSQL database is running and create the necessary tables. The application uses SQL migrations located in the `sql/schema/` directory. You'll need to run these migrations to set up your database schema.

## Usage

Once installed and configured, you can run `gator` with various commands:

### User Management

- **Register a new user**: `gator register <username>`
- **Login as a user**: `gator login <username>`
- **List all users**: `gator users`
- **Reset database** (deletes all users): `gator reset`

### Feed Management

- **Add a new feed**: `gator addfeed <name> <url>`
- **List all feeds**: `gator feeds`
- **Follow a feed**: `gator follow <url>`
- **List followed feeds**: `gator following`
- **Unfollow a feed**: `gator unfollow <url>`

### Aggregation

- **Start feed aggregation**: `gator agg <duration>` (e.g., `gator agg 1m` to fetch feeds every minute)
- **Browse posts**: `gator browse [limit]` (default limit is 2)

### Example Workflow

```bash
# Register a new user
gator register alice

# Add a feed
gator addfeed "Tech News" https://example.com/rss

# Follow the feed
gator follow https://example.com/rss

# Start aggregating feeds every 5 minutes
gator agg 5m

# In another terminal, browse your posts
gator browse 10
```

## Development

For development, you can use `go run .` to run the program directly:

```bash
go run . <command>
```

However, for production use, you should use the installed `gator` binary after running `go install` or `go build`.

## Notes

- Go programs are statically compiled binaries. After running `go build` or `go install`, you can run the `gator` binary without needing the Go toolchain installed.
- `go run .` is just for development. Use `gator` for production.

