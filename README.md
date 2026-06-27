# TrackerAPI

Tracks the mods for Questions for Staff, tickets, LOA, how many messages they send in a day, and the amount of time a mod spends in a VC. This will also allow Managers and above to read and edit LOA times. 

## Tech Stack

- **Go 1.22**
- **[discordgo](https://github.com/bwmarrin/discordgo)** — Discord API client
- **[pgx v5](https://github.com/jackc/pgx)** — PostgreSQL driver
- **[GORM](https://gorm.io/)** — ORM for database interactions
- **[godotenv](https://github.com/joho/godotenv)** — `.env` file loading

## Project Structure

```
TrackerAPI/
├── API/              # HTTP server and route handlers
├── core/             # Shared types, DB connection, utilities
├── modules/
│   ├── auth/         # Authentication
│   ├── dailymessages/# Daily message tracking
│   ├── loa/          # Leave of absence tracking
│   ├── qfs/          # QFS module
│   ├── tickets/      # Ticket tracking
│   └── voicetime/    # Voice channel time tracking
├── go.mod
├── go.sum
└── .gitignore
```

## Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- PostgreSQL database
- Discord Bot Token (from the [Discord Developer Portal](https://discord.com/developers/applications))

## Getting Started

**1. Clone the repository**

```bash
git clone https://github.com/brendanjhnsn/TrackerAPI.git
cd TrackerAPI
```

**2. Set up environment variables**

Create a `.env` file in the project root:

```env
DISCORD_TOKEN=your_discord_bot_token
DB_HOST=host
DB_PORT=port
DB_USER=your_db_user
DB_PASSWORD=your_db_password
DB_NAME=your_db_name
DB_NAME=
QFS_CHANNEL_ID=
MOD_ROLE_ID=
TICKET_CATEGORY_ID=
ADMIN_CATEGORY_ID=
MOD_ROLE_NAME=
QFS_CHANNEL_NAME=
DISCORD_CLIENT_ID=
DISCORD_CLIENT_SECRET=
DISCORD_GUILD_ID=
DISCORD_MANAGER_ROLE_ID=
DISCORD_DIRECTOR_ROLE_ID=
FRONTEND_URL=
```

**3. Install dependencies**

```bash
go mod tidy
```

**4. Run the server**

```bash
go run ./API
```


## Modules

| Module | Description |
|---|---|
| `auth` | Handles authentication and authorization |
| `dailymessages` | Tracks and stores daily message counts per user |
| `loa` | Manages leave of absence records |
| `qfs` | QFS tracking logic |
| `tickets` | Tracks which mod answer the ticket first |
| `voicetime` | Logs amount of time members spend in voice channels |

## Notes

- All modules are local Go modules referenced via `replace` directives in `go.mod`.
- MySQL driver is included as an indirect dependency but the primary database target is PostgreSQL via `pgx`.
