package postmark

// Attachment represents a file attachment included with an email message.
type Attachment struct {
	Name        string `json:"Name"`
	Content     string `json:"Content"`
	ContentType string `json:"ContentType"`
	ContentID   string `json:"ContentID,omitempty"`
}

// Header represents a custom email header included with a message.
type Header struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

// EmailResp is the response returned after sending a single email message.
type EmailResp struct {
	To          string `json:"To"`
	SubmittedAt string `json:"SubmittedAt"`
	MessageID   string `json:"MessageID"`
	ErrorCode   int    `json:"ErrorCode"`
	Message     string `json:"Message"`
}

// BatchEmailResp is the response returned after sending a batch of email messages.
// Each element corresponds to one message in the batch, in the same order.
type BatchEmailResp []EmailResp
