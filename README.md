# go-postmark

A Go client library for the [Postmark](https://postmarkapp.com) API, covering both administrative operations and email sending.

## Features

- **Email sending** — single, batch, templated, batch-with-templates, and async bulk jobs
- **Server management** — create, read, update, list, and delete Postmark Servers
- **Bounce management** — delivery stats, list/get bounces, activate bounced addresses, dump raw SMTP data
- **Message streams** — create, list, get, update, archive, and unarchive streams
- **Webhooks** — create, list, get, update, and delete webhook configurations
- **Domains** — create, list, get, update, delete, and verify (SPF / DKIM / Return-Path)
- **Sender Signatures** — create, list, get, update, delete, resend confirmation, verify SPF, rotate DKIM
- **Templates** — create, list, get, update, delete, validate, and push between servers
- **Outbound messages** — search, get details, dump raw message, open events, click events
- **Inbound messages** — search, get details, bypass rules, retry processing
- **Statistics** — outbound summary, send/bounce/spam/open/click counts by day, platform and client breakdowns
- **Inbound rules** — list, create, and delete trigger rules
- **Data removals** — submit and retrieve GDPR data removal requests

## Installation

```bash
go get github.com/tjsampson/go-postmark
```

## Authentication

Postmark uses two distinct token types. Both are configured via functional options on `New()`:

| Token | Option | Header sent | Used for |
|---|---|---|---|
| **Account token** | `APITokenOpt` | `X-Postmark-Account-Token` | Server management, domains, sender signatures, templates, webhooks, stats, inbound rules, data removals |
| **Server token** | `ServerTokenOpt` | `X-Postmark-Server-Token` | Email sending, bounces, message streams |

> **Important:** The account token is read automatically from the `POSTMARK_API_TOKEN` environment variable when no `APITokenOpt` is provided. There is **no** environment-variable default for the server token — you **must** pass `ServerTokenOpt` or every email/bounce/message-stream call will return an error immediately without making a network request.

Find your tokens in the Postmark UI:
- **Account token** → Account Settings → API Tokens
- **Server token** → [Your Server] → API Credentials

## Usage

### Creating a client

```go
import postmark "github.com/tjsampson/go-postmark"

// Reads account token from POSTMARK_API_TOKEN environment variable
client := postmark.New()

// Explicit tokens for both account-level and server-level operations
client = postmark.New(
    postmark.APITokenOpt("your-account-api-token"),
    postmark.ServerTokenOpt("your-server-api-token"),
)
```

### Sending email

#### Single email

```go
resp, err := client.SendEmail(&postmark.SendEmailReq{
    From:     "sender@example.com",
    To:       "recipient@example.com",
    Subject:  "Hello!",
    HtmlBody: "<p>Hello from go-postmark</p>",
    TextBody: "Hello from go-postmark",
    Tag:      "welcome",
})
if err != nil {
    log.Fatal(err)
}
fmt.Println("Sent message ID:", resp.MessageID)
```

#### Batch email

Postmark returns HTTP 200 even when individual messages fail. **You must inspect `ErrorCode` on every element** of the returned slice (0 = success).

```go
resps, err := client.SendBatch([]postmark.SendEmailReq{
    {From: "sender@example.com", To: "a@example.com", Subject: "Hi A", TextBody: "Hello A"},
    {From: "sender@example.com", To: "b@example.com", Subject: "Hi B", TextBody: "Hello B"},
})
if err != nil {
    log.Fatal(err)
}
for _, r := range resps {
    if r.ErrorCode != 0 {
        log.Printf("failed to deliver to %s: [%d] %s", r.To, r.ErrorCode, r.Message)
    }
}
```

#### Templated email

```go
trackOpens := true
resp, err := client.SendWithTemplate(&postmark.SendTemplateReq{
    From:          "sender@example.com",
    To:            "recipient@example.com",
    TemplateAlias: "welcome-email",
    TemplateModel: map[string]interface{}{
        "name":          "Alice",
        "action_url":    "https://example.com/confirm",
    },
    TrackOpens: &trackOpens,
})
if err != nil {
    log.Fatal(err)
}
```

You may identify the template by numeric ID or string alias:

```go
// By numeric ID
resp, err := client.SendWithTemplate(&postmark.SendTemplateReq{
    TemplateID: 12345,
    // ...
})

// By alias
resp, err = client.SendWithTemplate(&postmark.SendTemplateReq{
    TemplateAlias: "my-template",
    // ...
})
```

#### Batch with templates

Like `SendBatch`, partial success is a valid outcome. Inspect each `ErrorCode`.

```go
resps, err := client.SendBatchWithTemplates([]postmark.SendTemplateReq{
    {From: "sender@example.com", To: "a@example.com", TemplateAlias: "welcome-email", TemplateModel: map[string]interface{}{"name": "Alice"}},
    {From: "sender@example.com", To: "b@example.com", TemplateAlias: "welcome-email", TemplateModel: map[string]interface{}{"name": "Bob"}},
})
```

#### Bulk email jobs

For very large sends, use the async bulk job API:

```go
// Submit a bulk job (returns immediately with a job ID)
job, err := client.CreateBulkJob([]postmark.SendEmailReq{
    {From: "sender@example.com", To: "c@example.com", Subject: "Newsletter", TextBody: "..."},
    // thousands more...
})
if err != nil {
    log.Fatal(err)
}
fmt.Println("Bulk job ID:", job.ID, "Status:", job.Status)

// Poll for completion
status, err := client.GetBulkJob(job.ID)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Status: %s  Sent: %d  Errors: %d\n", status.Status, status.SuccessCount, status.ErrorCount)
```

### Attachments and inline images

```go
inlineCss := true
resp, err := client.SendEmail(&postmark.SendEmailReq{
    From:      "sender@example.com",
    To:        "recipient@example.com",
    Subject:   "Invoice",
    HtmlBody:  `<p>Please see attached.</p>`,
    Attachments: []postmark.EmailAttachment{
        {
            Name:        "invoice.pdf",
            ContentType: "application/pdf",
            Content:     base64EncodedPDFBytes, // base64-encoded string
        },
    },
})
```

### Server management

```go
// Create a server
server, err := client.CreateServer(&postmark.CreateServerReq{
    Name:  "Production",
    Color: "blue",
})

// Read a server
server, err = client.ReadServer("123")

// Update a server
updated, err := client.UpdateServer("123", &postmark.UpdateServerReq{
    Name:  "Production (v2)",
    Color: "green",
})

// List servers (page size 10, starting at offset 0)
list, err := client.ListServers("10", "0")
fmt.Println("Total:", list.TotalCount)

// Delete a server
del, err := client.DeleteServer("123")
fmt.Println(del.Message)
```

### Bounces

```go
// Overall delivery stats
stats, err := client.GetDeliveryStats()
fmt.Println("Inactive mails:", stats.InactiveMails)

// List bounces (all fields of GetBouncesParams are optional)
offset := 0
bounces, err := client.GetBounces(&postmark.GetBouncesParams{
    Count:  25,
    Offset: &offset,
    Type:   "HardBounce",
})

// Get a single bounce
bounce, err := client.GetBounce(bounceID)

// Get raw SMTP dump for a bounce
dump, err := client.GetBounceDump(bounceID)

// Reactivate a bounced address
activated, err := client.ActivateBounce(bounceID)

// List all bounce tags on this server
tags, err := client.GetBounceTags()
```

### Message streams

```go
// List all streams (optionally filter by type: "Transactional", "Inbound", "Broadcasts")
streams, err := client.ListMessageStreams("", false)

// Get a single stream
stream, err := client.GetMessageStream("outbound")

// Create a stream
stream, err = client.CreateMessageStream(&postmark.CreateMessageStreamReq{
    ID:                "my-broadcasts",
    Name:              "Marketing Broadcasts",
    MessageStreamType: "Broadcasts",
})

// Update a stream
stream, err = client.UpdateMessageStream("my-broadcasts", &postmark.UpdateMessageStreamReq{
    Name: "Marketing Broadcasts (2024)",
})

// Archive / unarchive
archived, err := client.ArchiveMessageStream("my-broadcasts")
stream, err = client.UnarchiveMessageStream("my-broadcasts")
```

### Webhooks

```go
// List webhooks (optionally filter by message stream)
webhooks, err := client.ListWebhooks("")

// Create a webhook
wh, err := client.CreateWebhook(&postmark.CreateWebhookReq{
    Url:           "https://example.com/hooks/postmark",
    MessageStream: "outbound",
    Triggers: &postmark.WebhookTriggers{
        Delivery: postmark.WebhookTriggerDelivery{Enabled: true},
        Bounce:   postmark.WebhookTriggerBounce{Enabled: true, IncludeContent: false},
        Open:     postmark.WebhookTriggerOpen{Enabled: true, PostFirstOpenOnly: true},
    },
})

// Get / update / delete
wh, err = client.GetWebhook(int64(wh.ID))
wh, err = client.UpdateWebhook(int64(wh.ID), &postmark.UpdateWebhookReq{
    Url: "https://example.com/hooks/postmark-v2",
})
del, err := client.DeleteWebhook(int64(wh.ID))
```

### Domains

```go
// List domains
list, err := client.ListDomains(50, 0)

// Create
domain, err := client.CreateDomain(&postmark.CreateDomainReq{Name: "example.com"})

// CRUD
domain, err = client.GetDomain(int64(domain.ID))
domain, err = client.UpdateDomain(int64(domain.ID), &postmark.UpdateDomainReq{
    ReturnPathDomain: "pm-bounces.example.com",
})
del, err := client.DeleteDomain(int64(domain.ID))

// Verify DNS records
domain, err = client.VerifyDomainDKIM(int64(domain.ID))
domain, err = client.VerifyDomainReturnPath(int64(domain.ID))
domain, err = client.VerifyDomainSPF(int64(domain.ID))
domain, err = client.RotateDomainDKIM(int64(domain.ID))
```

### Sender Signatures

```go
// List
sigs, err := client.ListSenderSignatures(50, 0)

// Create
sig, err := client.CreateSenderSignature(&postmark.CreateSenderSignatureReq{
    FromEmail: "noreply@example.com",
    Name:      "Example Notifications",
})

// CRUD
sig, err = client.GetSenderSignature(int64(sig.ID))
sig, err = client.UpdateSenderSignature(int64(sig.ID), &postmark.UpdateSenderSignatureReq{
    Name: "Example Notifications (updated)",
})
del, err := client.DeleteSenderSignature(int64(sig.ID))

// Confirmation / DNS actions
del, err = client.ResendSenderSignatureConfirmation(int64(sig.ID))
sig, err = client.VerifySenderSignatureSPF(int64(sig.ID))
sig, err = client.RequestNewDKIMForSenderSignature(int64(sig.ID))
```

### Templates

```go
// CRUD
tmpl, err := client.CreateTemplate(&postmark.CreateTemplateReq{
    Name:     "Welcome Email",
    Subject:  "Welcome, {{name}}!",
    HtmlBody: "<h1>Hi {{name}}</h1>",
    TextBody:  "Hi {{name}}",
    Alias:    "welcome-email",
})
tmpl, err = client.GetTemplate("welcome-email") // by alias or numeric ID string
tmpl, err = client.EditTemplate("welcome-email", &postmark.EditTemplateReq{
    Subject: "Welcome aboard, {{name}}!",
})
list, err := client.ListTemplates(50, 0)
del, err := client.DeleteTemplate("welcome-email")

// Validate template syntax without sending
inlineCss := true
result, err := client.ValidateTemplate(&postmark.ValidateTemplateReq{
    Subject:                    "Welcome, {{name}}!",
    HtmlBody:                   "<h1>Hi {{name}}</h1>",
    TextBody:                   "Hi {{name}}",
    TestRenderModel:            map[string]interface{}{"name": "Alice"},
    InlineCssForHtmlTestRender: &inlineCss,
})
fmt.Println("Valid:", result.AllContentIsValid)

// Push templates between servers (set PerformChanges=false for a dry run)
push, err := client.PushTemplate(&postmark.PushTemplateReq{
    SourceServerID:      100,
    DestinationServerID: 200,
    PerformChanges:      true,
})
```

### Outbound messages

```go
// Search
msgs, err := client.SearchOutboundMessages(postmark.OutboundMessageSearchParams{
    Count:     25,
    Offset:    0,
    Recipient: "alice@example.com",
    Status:    "Sent",
})

// Details and raw dump
details, err := client.GetOutboundMessageDetails(messageID)
dump, err := client.GetOutboundMessageDump(messageID)

// Open and click events
opens, err := client.GetOutboundMessageOpens(postmark.OutboundOpensParams{Count: 25, Offset: 0})
opens, err = client.GetOutboundMessageOpensByMessageID(messageID, 25, 0)
clicks, err := client.GetOutboundMessageClicks(postmark.OutboundClicksParams{Count: 25, Offset: 0})
clicks, err = client.GetOutboundMessageClicksByMessageID(messageID, 25, 0)
```

### Inbound messages

```go
// Search
msgs, err := client.SearchInboundMessages(postmark.InboundMessageSearchParams{
    Count:  25,
    Offset: 0,
    Status: "Processed",
})

// Details
details, err := client.GetInboundMessageDetails(messageID)

// Bypass processing rules / retry a failed message
bypass, err := client.BypassInboundMessageRules(messageID)
retry, err := client.RetryInboundMessage(messageID)
```

### Statistics

All stats endpoints accept a `StatsParams` struct with optional `Tag`, `FromDate`, `ToDate`, and `MessageStream` filters.

```go
params := postmark.StatsParams{
    FromDate: "2024-01-01",
    ToDate:   "2024-12-31",
}

// Summary
summary, err := client.GetOutboundStats(params)
fmt.Printf("Sent: %d  Bounced: %d  BounceRate: %.2f%%\n",
    summary.Sent, summary.Bounced, summary.BounceRate)

// Daily breakdowns
sends, err    := client.GetOutboundSendCounts(params)
bounces, err  := client.GetOutboundBounceCounts(params)
spam, err     := client.GetOutboundSpamCounts(params)
tracked, err  := client.GetOutboundTrackedEmailCounts(params)
opens, err    := client.GetOutboundOpenCounts(params)

// Platform / client / click breakdowns
platforms, err      := client.GetOutboundOpenPlatforms(params)
clients, err        := client.GetOutboundOpenEmailClients(params)
clicks, err         := client.GetOutboundClickCounts(params)
browsers, err       := client.GetOutboundClickBrowserFamilies(params)
clickPlatforms, err := client.GetOutboundClickPlatforms(params)
locations, err      := client.GetOutboundClickLocations(params)
```

### Inbound rules

```go
// List rules (paginated)
rules, err := client.ListInboundRules(25, 0)

// Create a block rule
rule, err := client.CreateInboundRule("spam@example.com")

// Delete a rule
del, err := client.DeleteInboundRule(rule.ID)
```

### Data removals (GDPR)

```go
// Submit a data removal request
removal, err := client.RequestDataRemoval(&postmark.DataRemovalReq{
    EmailAddress: "user@example.com",
    RequestedBy:  "privacy@yourcompany.com",
})
fmt.Println("Removal ID:", removal.ID, "Status:", removal.Status)

// Check status
removal, err = client.GetDataRemoval(removal.ID)
```

## Configuration Options

| Option | Description |
|---|---|
| `APITokenOpt(token string)` | Set the Postmark **account** API token explicitly. Defaults to `POSTMARK_API_TOKEN` env var. Used for account-level operations (server management, domains, webhooks, etc.). |
| `ServerTokenOpt(token string)` | Set the Postmark **server** API token. **Required** for email sending, bounces, and message streams. No env-var default. |
| `HTTPClientOpt(client *http.Client)` | Provide a custom `*http.Client` (e.g. for testing or custom TLS/proxy settings). |
| `TimeoutOpt(timeout time.Duration)` | Override the default 10-second HTTP request timeout. |

## Environment Variables

| Variable | Description |
|---|---|
| `POSTMARK_API_TOKEN` | Postmark **account** API token. Used as the default when `APITokenOpt` is not provided. |

## Error handling

The library surfaces Postmark API errors as `PostmarkErr` values. Two typed sentinels are provided for common cases:

```go
result, err := client.GetTemplate("missing-template")
if errors.Is(err, postmark.ErrNotFound) {
    // template does not exist
}

_, err = client.CreateDomain(&postmark.CreateDomainReq{Name: "already-exists.com"})
if errors.Is(err, postmark.ErrExists) {
    // domain already registered
}

// Inspect error code and message directly
var pmErr *postmark.PostmarkErr
if errors.As(err, &pmErr) {
    fmt.Printf("Postmark error %d: %s\n", pmErr.ErrorCode, pmErr.Message)
}
```

For **batch** email endpoints (`SendBatch`, `SendBatchWithTemplates`) Postmark returns HTTP 200 even when individual messages fail. The library does **not** aggregate per-message failures into a Go error because partial success is a valid outcome. Always inspect `ErrorCode` on each element of the returned slice.

## Helper functions

```go
// BoolPtr creates a *bool inline — useful for fields like TrackOpens and InlineCss
req := &postmark.SendEmailReq{
    TrackOpens: postmark.BoolPtr(false), // explicitly disable open tracking
}
```

## License

MIT
