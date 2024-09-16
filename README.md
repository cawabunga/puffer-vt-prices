# Puffer Validator Ticket Price History

This repository contains a Go application for aggregating Validator Ticket (VT) prices and a simple HTML page that renders the data as a table.

View the live price history: [https://cawabunga.github.io/puffer-vt-prices/](https://cawabunga.github.io/puffer-vt-prices/)

## Features

- Fetches and aggregates Puffer Validator Ticket prices
- Saves price data to a JSON file
- Displays price history in a web-based table format

## Prerequisites

- [Go](https://golang.org/doc/install)

## Usage

```bash
go run .
```

This command will:
1. Fetch the current VT prices
2. Aggregate the data
3. Save the results to `docs/events.json`

## Project Structure

- `main.go`: Contains the Go code for fetching and processing price data
- `docs/index.html`: Simple HTML page for rendering the price history table
- `docs/events.json`: JSON file storing the aggregated price data
