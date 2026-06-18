<!-- Postmark API client -->
# Go Postmark

A Go client library for Postmark administrative tasks — CRUD operations on Postmark Servers.

## Installation

```bash
go get github.com/tjsampson/go-postmark
```

## Usage

```go
package main

import (
    "fmt"
    "log"

    postmark "github.com/tjsampson/go-postmark"
)

func main() {
    // Create a client (reads POSTMARK_API_TOKEN from env by default)
    client := postmark.New()

    // Or provide an explicit API token
    client = postmark.New(postmark.APITokenOpt("your-account-api-token"))

    // Create a server
    server, err := client.CreateServer(&postmark.CreateServerReq{
        Name:             "My Server",
        Color:            "blue",
        SmtpApiActivated: true,
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created server: ID=%d Name=%s\n", server.ID, server.Name)

    // Read a server
    server, err = client.ReadServer(fmt.Sprintf("%d", server.ID))
    if err != nil {
        log.Fatal(err)
    }

    // Update a server
    updated, err := client.UpdateServer(fmt.Sprintf("%d", server.ID), &postmark.UpdateServerReq{
        Name:  "Renamed Server",
        Color: "green",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Updated server: %s\n", updated.Name)

    // List servers (count=10, offset=0)
    list, err := client.ListServers("10", "0")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Total servers: %d\n", list.TotalCount)

    // Delete a server
    del, err := client.DeleteServer(fmt.Sprintf("%d", server.ID))
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(del.Message)
}
```

## Configuration Options

| Option | Description |
|---|---|
| `APITokenOpt(token string)` | Set the Postmark account API token explicitly |
| `HTTPClientOpt(client *http.Client)` | Provide a custom `*http.Client` (e.g. for testing) |
| `TimeoutOpt(timeout time.Duration)` | Override the default 10-second HTTP timeout |

## Environment Variables

| Variable | Description |
|---|---|
| `POSTMARK_API_TOKEN` | Postmark account API token (used when no `APITokenOpt` is provided) |

## License

MIT
