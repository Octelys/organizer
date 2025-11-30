# Magazine Organizer

A Go application that uses the OpenAI API to automatically organize and process scanned magazine pages. The application scans directories of magazine page images, analyzes them using AI to extract metadata, and organizes the files into a structured output directory.

The program follows a concurrent, service-oriented architecture with three main components that communicate via channels:

1. **Scanner Service**: Scans directories for magazine page images and uses AI to determine the correct page order.
2. **Analyzer Service**: Analyzes cover pages using vision AI to extract metadata (title, publication number, month, year).
3. **Copier Service**: Organizes and copies the files to the output directory with proper naming and structure.

All services run concurrently using goroutines and communicate through typed channels.

## Prerequisites

- Go 1.25+ (or the version specified in `go.mod`)
- An OpenAI API key with access to GPT-4 or GPT-5 models (for vision capabilities)
- Image files (JPEG/PNG) of magazine pages organized in subdirectories

## Configuration

The program is configured entirely via environment variables:

- `OPENAI_API_KEY` (required): Your OpenAI API key.
- `WORKING_DIR` (required): Absolute or relative path to the directory containing subdirectories of magazine page images.
- `OUTPUT_DIR` (required): Absolute or relative path where organized magazines will be output.

In GoLand, you can set these in **Run | Edit Configurations...** under **Environment variables**.

## Installation

Clone the repository and download Go dependencies:

```bash
git clone <your-repo-url> organizer
cd organizer
go mod tidy
```

## Running the program

### Using Make (recommended)

The project includes a Makefile for common tasks:

```bash
# Build the application
make build

# Run from source
export OPENAI_API_KEY="your-api-key-here"
export WORKING_DIR="/path/to/your/magazine/scans"
export OUTPUT_DIR="/path/to/output/directory"
make run

# Clean build artifacts
make clean

# Run tests, linters, and build
make all

# Show all available commands
make help
```

### Building the application manually

```bash
go build -o bin/organizer ./cmd/organizer
```

### Running from source

```bash
export OPENAI_API_KEY="your-api-key-here"
export WORKING_DIR="/path/to/your/magazine/scans"
export OUTPUT_DIR="/path/to/output/directory"

go run ./cmd/organizer
```

### Running the compiled binary

```bash
export OPENAI_API_KEY="your-api-key-here"
export WORKING_DIR="/path/to/your/magazine/scans"
export OUTPUT_DIR="/path/to/output/directory"

./bin/organizer
```

### From GoLand

1. Create or edit a **Run/Debug Configuration** for `cmd/organizer/main.go`.
2. Set `OPENAI_API_KEY`, `WORKING_DIR`, and `OUTPUT_DIR` in **Environment variables**.
3. Run the configuration.

## How it works

The application uses a concurrent pipeline architecture with three services:

### 1. Scanner Service

- Scans each subdirectory in `WORKING_DIR` for image files
- Sends filenames to OpenAI to determine the correct page order
- Produces `MagazinePages` objects containing ordered page information
- Sends results through a channel to the Analyzer Service

### 2. Analyzer Service

- Receives `MagazinePages` from the Scanner via a channel
- Identifies the cover page (typically the first page)
- Uses OpenAI vision API to analyze the cover image and extract:
  - Magazine title
  - Publication number
  - Month(s) of publication
  - Year of publication
- Produces `Magazine` objects with complete metadata
- Sends results through a channel to the Copier Service

### 3. Copier Service

- Receives `Magazine` objects from the Analyzer via a channel
- Creates organized directory structure in `OUTPUT_DIR`
- Copies and renames files according to the extracted metadata
- Format: `{Title}/{Year}/{Number} - {Months}/page_{n}.jpg`

### Concurrency Model

All three services run as goroutines that communicate via typed channels:

- `chan entities.MagazinePages`: Scanner → Analyzer
- `chan entities.Magazine`: Analyzer → Copier

The main goroutine uses a shared `sync.WaitGroup` to wait for all services to complete processing.

### Additional Services

- **Configuration Service**: Manages environment variables and application settings
- **AI Proxy**: Wraps the OpenAI API client with convenience methods for text and vision requests
- **Audit Service**: Logs processing events and errors to timestamped audit files

## Project Structure

```
organizer/
├── cmd/
│   └── organizer/
│       └── main.go                  # Application entry point
├── internal/
│   ├── abstractions/
│   │   ├── entities/                # Domain entities (Magazine, MagazinePages, etc.)
│   │   └── interfaces/              # Channel and service interfaces
│   ├── ai/                          # OpenAI API client wrapper
│   ├── analyzer/                    # Cover page analysis service
│   ├── audit/                       # Audit logging service
│   ├── configuration/               # Configuration management
│   ├── copier/                      # File organization and copying service
│   └── scanner/                     # Directory scanning and page ordering service
├── bin/                             # Compiled binaries (gitignored)
├── Makefile                         # Build automation
├── go.mod                           # Module definition and dependencies
├── go.sum                           # Dependency checksums
└── README.md
```

## License

Add your preferred license here (e.g., MIT).
