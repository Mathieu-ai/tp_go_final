# URL Shortener - Architecture & Logic Documentation

## 📋 Table of Contents

- [Application Overview](#application-overview)
- [Link Creation Logic](#link-creation-logic)
- [Click Tracking Logic](#click-tracking-logic)
- [Data Models](#data-models)
- [Method Call Sequences](#method-call-sequences)
- [File Relationships](#file-relationships)
- [Commands to Run](#commands-to-run)

## 🔍 Application Overview

This URL shortener application is built with Go using a clean architecture pattern with the following layers:

- **API Layer**: HTTP handlers (Gin framework)
- **Service Layer**: Business logic
- **Repository Layer**: Data access abstraction
- **Models**: Data structures and database schemas
- **Workers**: Asynchronous processing
- **Monitor**: URL health checking

## 🔗 Link Creation Logic

### Flow Diagram

```
User Request → API Handler → Link Service → Repository → Database
     ↓              ↓            ↓            ↓           ↓
  JSON Body    Validation   Code Generation  GORM ORM   SQLite
```

### Detailed Process

1. **HTTP Request** (`POST /api/v1/links`)
   - Single URL: `{"long_url": "https://example.com"}`
   - Multiple URLs: `{"long_urls": ["https://example.com", "https://google.com"]}`
   - Handled by `CreateShortLinkHandler()` in `internal/api/handlers.go`

2. **CLI Request** (`./url-shortener create`)
   - Single URL: `--url="https://example.com"`
   - Multiple URLs as JSON: `--url='["https://example.com", "https://google.com"]'`
   - Multiple URLs with single quotes: `--url="['https://example.com','https://google.com']"`
   - Handled by `CreateCmd` in `cmd/cli/create.go` with flexible `parseURLFlag()` function

3. **Validation**
   - API: Gin binding validates URL format using `binding:"omitempty,url"` for single URL
   - API: Multiple URLs use `binding:"omitempty,dive,url"` to validate each URL in array
   - CLI: `url.ParseRequestURI()` validates each parsed URL before processing
   - Request body parsed into `CreateLinkRequest` struct (API) or parsed by `parseURLFlag()` (CLI)

4. **Service Layer Processing**
   - Single URL: `linkService.CreateLink(req.LongURL)` called once
   - Multiple URLs: `linkService.CreateLink()` called for each URL in loop
   - `GenerateShortCode(6)` creates cryptographically secure random code for each
   - Collision detection with retry logic (max 5 attempts per URL)

5. **Database Storage**
   - Repository creates `Link` model with unique short code for each URL
   - GORM persists each link to SQLite database

6. **Response**
   - API Single URL: Returns JSON with `short_code`, `long_url`, and `full_short_url`
   - API Multiple URLs: Returns array of results with summary statistics
   - CLI: Displays formatted output with success/failure status for each URL

### Key Methods Called

```go
// API Layer (Single URL)
CreateShortLinkHandler() → handleSingleURL() → linkService.CreateLink()

// API Layer (Multiple URLs)  
CreateShortLinkHandler() → handleMultipleURLs() → linkService.CreateLink() (for each URL)

// CLI Layer (Single/Multiple URLs)
CreateCmd.Run() → parseURLFlag() → linkService.CreateLink() (for each URL)

// Service Layer  
linkService.CreateLink() → GenerateShortCode() → linkRepo.GetLinkByShortCode() → linkRepo.CreateLink()

// Repository Layer
linkRepo.CreateLink() → db.Create()
```

## 📊 Click Tracking Logic

### Asynchronous Click Processing Flow

```
URL Access → Redirect Handler → Click Event → Channel → Workers → Database
     ↓            ↓               ↓           ↓         ↓         ↓
  Browser     Find Link      Create Event   Buffer   Goroutines SQLite
```

### Detailed Process

1. **URL Access** (`GET /{shortCode}`)
   - User clicks short URL (e.g., `http://localhost:8080/abc123`)
   - Handled by `RedirectHandler()` in `internal/api/handlers.go`

2. **Link Lookup**
   - `linkService.GetLinkByShortCode(shortCode)` finds target URL
   - Returns 404 if short code not found

3. **Click Event Creation**
   - Creates `ClickEvent` struct with:
     - `LinkID`: Database ID of the clicked link
     - `Timestamp`: Current time
     - `UserAgent`: Browser information
     - `IPAddress`: Client IP address

4. **Asynchronous Processing**
   - Event sent to buffered channel (`ClickEventsChannel`)
   - Non-blocking operation using `select` statement
   - If channel full, event is dropped (logged as warning)

5. **Immediate Redirect**
   - HTTP 302 redirect to original URL
   - User sees no delay regardless of analytics processing

6. **Background Workers**
   - Pool of goroutines process click events
   - Convert `ClickEvent` to `Click` model
   - Save to database via `clickRepo.CreateClick()`

### Key Methods Called

```go
// API Layer
RedirectHandler() → linkService.GetLinkByShortCode() → Channel Send → c.Redirect()

// Worker Layer
StartClickWorkers() → clickWorker() → clickRepo.CreateClick()

// Repository Layer
clickRepo.CreateClick() → db.Create()
```

## 📋 Data Models

### Link Model

```go
type Link struct {
    ID        uint      `gorm:"primaryKey"`
    ShortCode string    `gorm:"uniqueIndex;size:10;not null"`
    LongURL   string    `gorm:"not null"`
    CreatedAt time.Time `gorm:"autoCreateTime"`
}
```

### Click Model

```go
type Click struct {
    ID        uint      `gorm:"primaryKey"`
    LinkID    uint      `gorm:"index"`           // Foreign key to Link
    Link      Link      `gorm:"foreignKey:LinkID"`
    Timestamp time.Time
    UserAgent string    `gorm:"size:255"`
    IPAddress string    `gorm:"size:50"`
}
```

### ClickEvent (Channel Communication)

```go
type ClickEvent struct {
    LinkID    uint
    Timestamp time.Time
    UserAgent string
    IPAddress string
}
```

### API Request/Response Models

```go
// CreateLinkRequest - API request body
type CreateLinkRequest struct {
    LongURL  string   `json:"long_url" binding:"omitempty,url"`
    LongURLs []string `json:"long_urls" binding:"omitempty,dive,url"`
}

// CreateLinkResponse - Single link response
type CreateLinkResponse struct {
    ShortCode    string `json:"short_code"`
    LongURL      string `json:"long_url"`
    FullShortURL string `json:"full_short_url"`
    Success      bool   `json:"success"`
    Error        string `json:"error,omitempty"`
}

// CreateLinksResponse - Multiple links response
type CreateLinksResponse struct {
    Results []CreateLinkResponse `json:"results"`
    Summary struct {
        Total      int `json:"total"`
        Successful int `json:"successful"`
        Failed     int `json:"failed"`
    } `json:"summary"`
}
```

## 📈 Method Call Sequences

### 1. Link Creation Sequence (CLI)

```
CLI: create.go
├── parseURLFlag() (handles JSON arrays and single URLs)
├── config.LoadConfig()
├── gorm.Open()
├── repository.NewLinkRepository()
├── services.NewLinkService()
└── For each URL:
    ├── linkService.CreateLink()
    │   ├── GenerateShortCode()
    │   ├── linkRepo.GetLinkByShortCode() (collision check)
    │   └── linkRepo.CreateLink()
    └── Display result
```

### 2. Link Creation Sequence (API)

```
API: handlers.go
├── CreateShortLinkHandler()
├── c.ShouldBindJSON()
├── handleSingleURL() OR handleMultipleURLs()
├── linkService.CreateLink() (for each URL)
└── c.JSON() (response)
```

### 3. URL Redirection Sequence

```
handlers.go: RedirectHandler()
├── c.Param("shortCode")
├── linkService.GetLinkByShortCode()
├── Create ClickEvent
├── Channel Send (non-blocking)
└── c.Redirect() (HTTP 302)

Background Workers:
├── StartClickWorkers() (launches worker pool)
├── clickWorker() (goroutine)
├── Range over channel
├── Convert ClickEvent → Click
└── clickRepo.CreateClick()
```

### 4. Statistics Retrieval

```
CLI: stats.go
├── config.LoadConfig()
├── gorm.Open()
├── linkService.GetLinkStats()
├── linkRepo.GetLinkByShortCode()
├── linkRepo.CountClicksByLinkID()
└── Display results

API: handlers.go
├── GetLinkStatsHandler()
├── linkService.GetLinkStats()
└── c.JSON() (response)
```

### 5. Database Migration

```
CLI: migrate.go
├── config.LoadConfig()
├── gorm.Open()
├── db.AutoMigrate(&models.Link{}, &models.Click{})
└── Success message
```

### 6. Server Startup

```
server.go: RunServerCmd
├── config.LoadConfig()
├── gorm.Open()
├── db.AutoMigrate()
├── Initialize repositories and services
├── make(chan models.ClickEvent, bufferSize)
├── workers.StartClickWorkers()
├── monitor.NewUrlMonitor() → go urlMonitor.Start()
├── api.SetupRoutes()
├── srv.ListenAndServe()
└── Graceful shutdown handling
```

## 🏗️ File Relationships & Architecture

### Schematic Overview

```
main.go
└── cmd/
    ├── root.go (Cobra setup)
    ├── server/server.go (HTTP server + workers)
    └── cli/
        ├── create.go (CLI link creation with multi-URL support)
        ├── stats.go (CLI statistics)
        └── migrate.go (Database setup)

internal/
├── api/handlers.go (HTTP routes & handlers)
├── services/
│   ├── link_service.go (Business logic)
│   └── click_service.go (Click operations)
├── repository/
│   ├── link_repository.go (Data access interface)
│   └── click_repository.go (Click data access)
├── models/
│   ├── link.go (Database models)
│   └── click.go (Click & ClickEvent structs)
├── workers/click_worker.go (Async processing)
├── monitor/url_monitor.go (Health checking)
├── config/config.go (Configuration management)
└── errors/errors.go (Custom error types)

configs/config.yaml (Application settings)
```

### Dependency Flow

```
Handlers → Services → Repositories → Models → Database
    ↓         ↓           ↓           ↓         ↓
   Gin    Business    Interface   GORM     SQLite
          Logic      Abstraction  ORM
```

### Component Interactions

#### API Layer (`internal/api/`)

- **Purpose**: HTTP request handling, routing, JSON serialization
- **Dependencies**: Services, Models
- **Interactions**: Receives requests, validates input, calls services, returns responses

#### Service Layer (`internal/services/`)

- **Purpose**: Business logic, validation, orchestration
- **Dependencies**: Repositories, Models, Errors
- **Interactions**: Processes business rules, manages transactions, handles errors

#### Repository Layer (`internal/repository/`)

- **Purpose**: Data access abstraction, database operations
- **Dependencies**: Models, GORM
- **Interactions**: CRUD operations, query building, data mapping

#### Worker Layer (`internal/workers/`)

- **Purpose**: Asynchronous background processing
- **Dependencies**: Repositories, Models
- **Interactions**: Processes channel messages, bulk operations

#### Monitor Layer (`internal/monitor/`)

- **Purpose**: URL health checking, status notifications
- **Dependencies**: Repositories, HTTP client
- **Interactions**: Periodic checks, state tracking, logging

## 🚀 Commands to Run

### Prerequisites

```bash
# Clone the repository
git clone https://github.com/axellelanca/urlshortener.git
cd urlshortener

# Download dependencies
go mod tidy
```

### Build Application

```bash
# Compile the application
go build -o url-shortener
```

### Database Setup (REQUIRED FIRST)

```bash
# Create database tables
./url-shortener migrate
```

### Start the Server

```bash
# Launch HTTP server + background workers + URL monitor
./url-shortener run-server
```

### CLI Usage (New Terminal Window)

```bash
# Create a single short URL
./url-shortener create --url="https://www.google.com"

# Create multiple URLs using JSON array format
./url-shortener create --url='["https://www.google.com", "https://www.github.com", "https://www.stackoverflow.com"]'

# Create multiple URLs using single quotes (for shell compatibility)
./url-shortener create --url="['https://www.google.com','https://www.github.com']"

# Get statistics for a short code
./url-shortener stats --code="abc123"
```

### API Usage (Alternative to CLI)

```bash
# Health check
curl http://localhost:8080/health

# Create single short URL via API (backward compatible)
curl -X POST http://localhost:8080/api/v1/links \
  -H "Content-Type: application/json" \
  -d '{"long_url":"https://www.example.com"}'

# Create multiple short URLs via API (new feature)
curl -X POST http://localhost:8080/api/v1/links \
  -H "Content-Type: application/json" \
  -d '{"long_urls":["https://www.example.com", "https://www.google.com", "https://www.github.com"]}'

# Get statistics via API
curl http://localhost:8080/api/v1/links/abc123/stats

# Test redirection (in browser)
# Visit: http://localhost:8080/abc123
```

### API Response Formats

#### Single URL Response (Backward Compatible)

```json
{
  "short_code": "abc123",
  "long_url": "https://www.example.com",
  "full_short_url": "http://localhost:8080/abc123"
}
```

#### Multiple URLs Response (New Format)

```json
{
  "results": [
    {
      "short_code": "abc123",
      "long_url": "https://www.example.com",
      "full_short_url": "http://localhost:8080/abc123",
      "success": true
    },
    {
      "short_code": "def456",
      "long_url": "https://www.google.com",
      "full_short_url": "http://localhost:8080/def456",
      "success": true
    },
    {
      "long_url": "invalid-url",
      "success": false,
      "error": "Failed to create short link"
    }
  ],
  "summary": {
    "total": 3,
    "successful": 2,
    "failed": 1
  }
}
```

### Testing the System

```bash
# 1. Start server (Terminal 1)
./url-shortener run-server

# 2. Create single URL (Terminal 2)
./url-shortener create --url="https://www.google.com"
# Output: Code: xyz123, Full URL: http://localhost:8080/xyz123

# 3. Create multiple URLs (Terminal 2)
./url-shortener create --url='["https://www.google.com", "https://www.github.com"]'

# 4. Test redirection (Browser)
# Visit: http://localhost:8080/xyz123
# Should redirect to https://www.google.com

# 5. Check statistics (Terminal 2)
./url-shortener stats --code="xyz123"
# Should show: Total clicks: 1 (or more)
```

### Shutdown

```bash
# Stop server gracefully
Ctrl + C  # In the terminal running run-server
```

## 🔧 Configuration

The application uses `configs/config.yaml` for settings:

```yaml
server:
  port: 8080
  base_url: "http://localhost:8080"
database:
  name: "url_shortener.db"
analytics:
  buffer_size: 1000    # Click event channel buffer
  worker_count: 5      # Background worker goroutines
monitor:
  interval_minutes: 5  # URL health check frequency
```

Environment variables can override config values:

```bash
export SERVER_PORT=9090
export DATABASE_NAME=custom.db
export ANALYTICS_WORKER_COUNT=10
```

## 🎯 Key Features

- **Flexible CLI Input**: Support for single URLs and JSON arrays with multiple quote formats
- **Backward Compatible API**: Maintains existing single URL format while adding multiple URL support
- **Non-blocking Redirects**: Click tracking never delays URL redirection
- **Collision Handling**: Automatic retry for duplicate short codes
- **Health Monitoring**: Periodic URL accessibility checking
- **Graceful Shutdown**: Clean termination of background processes
- **Configurable**: Environment variables and YAML configuration
- **Scalable**: Worker pool pattern for high-volume click processing
- **Comprehensive Error Handling**: Detailed error messages and partial success support
