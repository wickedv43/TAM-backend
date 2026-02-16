package common

type Notification struct {
	UserID    int64
	Text      string
	WebAppURL string // Optional: URL for inline button
}
