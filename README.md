# go-rename

A small Go utility that uses the OpenAI API to infer the correct ordering of page image files in a directory (for example, magazine pages), based on their filenames.

The program:

1. Reads all file names from a working directory.
2. Asks an OpenAI model to guess the correct order of those files based on page numbers in the names.
3. Parses the JSON array of ordered file names returned by the model.

This repository is a minimal example meant to be edited and extended in GoLand.

## Prerequisites

- Go 1.22+ (or the version specified in `go.mod`)
- An OpenAI API key with access to the model you configure in `main.go`

## Configuration

The program is configured entirely via environment variables:

- `OPENAI_API_KEY` (required): Your OpenAI API key.
- `WORKING_DIR` (required): Absolute or relative path to the directory whose files you want to analyze.

In GoLand, you can set these in **Run | Edit Configurations...** under **Environment variables**.

## Installation

Clone the repository and download Go dependencies:

```bash
git clone <your-repo-url> go-rename
cd go-rename
go mod tidy
```

## Running the program

From the command line:

```bash
export OPENAI_API_KEY="your-api-key-here"
export WORKING_DIR="/path/to/your/files"

go run ./...
```

From GoLand:

1. Create or edit a **Run/Debug Configuration** for `main.go`.
2. Set `OPENAI_API_KEY` and `WORKING_DIR` in **Environment variables**.
3. Run the configuration.

## How it works

The core logic is in `main.go`:

1. Read `OPENAI_API_KEY` and `WORKING_DIR` from the environment.
2. Use `os.ReadDir` to list all files in `WORKING_DIR` and print them.
3. Build a natural-language prompt enumerating those filenames.
4. Call the OpenAI Chat Completions and Responses APIs via the official Go SDK (`github.com/openai/openai-go/v3`).
5. Expect the model to respond with a JSON array (e.g. `["page_01.png", "page_02.png", ...]`) describing the correct order.
6. Use `encoding/json` to unmarshal the JSON string into a `[]string` slice (`orderedFiles`).
7. (Optional extension) Upload the cover image and call the Responses API with the image attached to infer a publication month/year.

You can then extend the program to:

- Actually rename or move files according to `orderedFiles`.
- Extract additional metadata (e.g., publication month/year) from the cover page.
- Handle errors and edge cases more robustly.

## License

Add your preferred license here (e.g., MIT).
