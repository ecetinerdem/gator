# Gator - RSS Feed Aggregator

A command-line RSS feed aggregator built with Go, PostgreSQL, and sqlc. Gator allows you to follow multiple RSS feeds, automatically fetch new posts, and browse them directly in your terminal.

## Features

- 📰 **Multi-user support** - Multiple users can manage their own feed subscriptions
- 🔄 **Automatic aggregation** - Continuously fetches posts from followed feeds
- 👤 **User authentication** - Login system to manage your personal feed list
- 📚 **Feed management** - Add, follow, unfollow, and view feeds
- 📖 **Browse posts** - View posts from your followed feeds in the terminal
- 🗄️ **PostgreSQL database** - Reliable data storage with proper foreign key relationships

## Prerequisites

- Go 1.21 or higher
- PostgreSQL 14 or higher
- [goose](https://github.com/pressly/goose) - Database migration tool
- [sqlc](https://github.com/sqlc-dev/sqlc) - SQL code generator

## Installation

1. Clone the repository:
```bash
git clone https://github.com/ecetinerdem/gator.git
cd gator
```

2. Install dependencies:
```bash
go mod download
```

3. Install goose and sqlc:
```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

4. Set up PostgreSQL database:
```bash
createdb gator
```

5. Run migrations:
```bash
goose -dir sql/schema postgres "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable" up
```

6. Generate SQL code:
```bash
sqlc generate
```

7. Create config file in your home directory (`~/.gatorconfig.json`):
```json
{
  "db_url": "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable",
  "current_user_name": ""
}
```

8. Build the project:
```bash
go build -o gator
```

## Usage

### User Management

**Register a new user:**
```bash
./gator register <username>
```

**Login as a user:**
```bash
./gator login <username>
```

**View all users:**
```bash
./gator users
```

### Feed Management

**Add a new feed:**
```bash
./gator addfeed <feed_name> <feed_url>
```
Example:
```bash
./gator addfeed "TechCrunch" "https://techcrunch.com/feed/"
```

**View all feeds:**
```bash
./gator feeds
```

**Follow an existing feed:**
```bash
./gator follow <feed_url>
```

**Unfollow a feed:**
```bash
./gator unfollow <feed_url>
```

**View feeds you're following:**
```bash
./gator following
```

### Aggregation & Browsing

**Start the aggregator (runs continuously):**
```bash
./gator agg <duration>
```
Example:
```bash
./gator agg 1m    # Fetches feeds every 1 minute
./gator agg 30s   # Fetches feeds every 30 seconds
./gator agg 1h    # Fetches feeds every 1 hour
```

**Browse posts from followed feeds:**
```bash
./gator browse [limit]
```
Examples:
```bash
./gator browse      # Shows 2 posts (default)
./gator browse 10   # Shows 10 posts
```

### Maintenance

**Reset database (delete all users):**
```bash
./gator reset
```

## Project Structure

```
gator/
├── main.go                      # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go           # Configuration and command handlers
│   └── database/
│       ├── db.go               # Database connection
│       ├── models.go           # Generated database models
│       ├── users.sql.go        # Generated user queries
│       ├── feeds.sql.go        # Generated feed queries
│       ├── feed_follows.sql.go # Generated feed follow queries
│       └── posts.sql.go        # Generated post queries
├── sql/
│   ├── schema/                 # Database migrations
│   │   ├── 001_users.sql
│   │   ├── 002_feeds.sql
│   │   ├── 003_feed_follows.sql
│   │   ├── 004_feeds_last_fetched.sql
│   │   └── 005_posts.sql
│   └── queries/                # SQL queries for sqlc
│       ├── users.sql
│       ├── feeds.sql
│       ├── feed_follows.sql
│       └── posts.sql
└── sqlc.yaml                   # sqlc configuration
```

## Database Schema

### Tables

- **users** - User accounts
- **feeds** - RSS feed URLs and metadata
- **feed_follows** - Many-to-many relationship between users and feeds
- **posts** - Individual posts from RSS feeds

### Relationships

- Each feed belongs to a user (creator)
- Users can follow multiple feeds
- Posts belong to feeds
- When a user or feed is deleted, related records cascade

## Popular RSS Feeds to Try

```bash
# Tech News
./gator addfeed "TechCrunch" "https://techcrunch.com/feed/"
./gator addfeed "Hacker News" "https://news.ycombinator.com/rss"
./gator addfeed "The Verge" "https://www.theverge.com/rss/index.xml"

# Development
./gator addfeed "Boot.dev Blog" "https://blog.boot.dev/index.xml"
./gator addfeed "Go Blog" "https://go.dev/blog/feed.atom"

# General News
./gator addfeed "BBC News" "http://feeds.bbci.co.uk/news/rss.xml"
./gator addfeed "NPR" "https://feeds.npr.org/1001/rss.xml"
```

## Workflow Example

```bash
# 1. Register and login
./gator register john
./gator login john

# 2. Add some feeds
./gator addfeed "TechCrunch" "https://techcrunch.com/feed/"
./gator addfeed "Hacker News" "https://news.ycombinator.com/rss"

# 3. View all feeds
./gator feeds

# 4. Start aggregator in background (separate terminal)
./gator agg 1m

# 5. Browse posts (in another terminal)
./gator browse 5

# 6. Follow more feeds
./gator follow "https://blog.boot.dev/index.xml"
./gator following

# 7. Unfollow a feed
./gator unfollow "https://news.ycombinator.com/rss"
```

## Configuration

The configuration file is stored at `~/.gatorconfig.json`:

```json
{
  "db_url": "postgres://username:password@localhost:5432/gator?sslmode=disable",
  "current_user_name": "john"
}
```

- **db_url** - PostgreSQL connection string
- **current_user_name** - Currently logged-in user (set automatically by login command)

## Development

### Running Migrations

Create a new migration:
```bash
goose -dir sql/schema create <migration_name> sql
```

Run migrations:
```bash
goose -dir sql/schema postgres "your_connection_string" up
```

Roll back migrations:
```bash
goose -dir sql/schema postgres "your_connection_string" down
```

### Regenerating Database Code

After modifying SQL queries:
```bash
sqlc generate
```

## Architecture Patterns

### Middleware Pattern
Commands requiring authentication use middleware to avoid code duplication:
```go
cmds.Register("browse", config.MiddlewareLoggedIn(config.BrowseHandler))
```

### Command Pattern
All commands follow a consistent interface:
```go
func Handler(s *State, cmd Command) error
```

### Repository Pattern
Database operations are abstracted through generated sqlc code, providing type-safe database queries.

## Error Handling

- Duplicate feed URLs are handled gracefully
- Missing users return appropriate error messages
- Database connection errors are logged
- RSS parsing errors are caught and logged without stopping aggregation

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is open source and available under the MIT License.

## Acknowledgments

- Built as part of the [Boot.dev](https://boot.dev) curriculum
- Uses [goose](https://github.com/pressly/goose) for migrations
- Uses [sqlc](https://github.com/sqlc-dev/sqlc) for type-safe SQL
- RSS parsing inspired by RSS 2.0 specification

## Troubleshooting

### Database connection errors
- Verify PostgreSQL is running: `pg_isready`
- Check connection string in `~/.gatorconfig.json`
- Ensure database exists: `psql -l`

### Migration errors
- Check migration status: `goose -dir sql/schema postgres "connection_string" status`
- Reset migrations if needed: `goose -dir sql/schema postgres "connection_string" reset`

### Feed fetching issues
- Verify feed URLs are accessible in browser
- Check for rate limiting (increase time between requests)
- Some feeds may have non-standard date formats

## Support

For issues, questions, or contributions, please open an issue on GitHub.

---

**Happy aggregating! 📰**

## 🧑‍💻 Author

**E. Çetin Erdem**  
[GitHub: @ecetinerdem](https://github.com/ecetinerdem)

---

