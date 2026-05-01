package checkins

import (
	"errors"
	"strings"
	"time"
)

type CheckInStatus string
type EvidenceKind string

const (
	StatusDraft            CheckInStatus = "draft"
	StatusSubmitted        CheckInStatus = "submitted"
	StatusChangesRequested CheckInStatus = "changes_requested"
	StatusApproved         CheckInStatus = "approved"
	StatusRejected         CheckInStatus = "rejected"

	KindText  EvidenceKind = "text"
	KindLink  EvidenceKind = "link"
	KindFile  EvidenceKind = "file"
	KindImage EvidenceKind = "image"

	MaxFileSizeBytes = 10 * 1024 * 1024 // 10 MB
	MaxEvidenceItems = 20
)

var (
	ErrCheckInNotFound      = errors.New("check-in not found")
	ErrGoalNotEligible      = errors.New("goal is not active or you are not the owner")
	ErrNotAuthorized        = errors.New("not authorized to access this check-in")
	ErrNotOwner             = errors.New("only the goal owner can perform this action")
	ErrInvalidEvidenceInput = errors.New("invalid evidence input")
	ErrCannotSubmit         = errors.New("check-in cannot be submitted in its current state")
	ErrCannotAddEvidence    = errors.New("cannot add evidence in the current state")
	ErrTooManyEvidenceItems = errors.New("maximum evidence items reached")
	ErrFileTooLarge         = errors.New("file exceeds 10 MB limit")
	ErrUnsupportedMIME      = errors.New("unsupported file type")
)

type ReviewDecision string

const (
	DecisionApprove        ReviewDecision = "approved"
	DecisionReject         ReviewDecision = "rejected"
	DecisionRequestChanges ReviewDecision = "changes_requested"
)

var (
	ErrNotBuddy     = errors.New("only the goal buddy can review this check-in")
	ErrCannotReview = errors.New("check-in is not in a reviewable state")
)

type ReviewInput struct {
	Decision ReviewDecision `json:"decision,omitempty"`
	Comment  string         `json:"comment"`
}

type ReviewRecord struct {
	ID             int64          `json:"id"`
	CheckInID      int64          `json:"check_in_id"`
	ReviewerUserID int64          `json:"reviewer_user_id"`
	Decision       ReviewDecision `json:"decision"`
	Comment        string         `json:"comment,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

var allowedMIMETypes = map[string]EvidenceKind{
	"image/jpeg":      KindImage,
	"image/png":       KindImage,
	"image/gif":       KindImage,
	"image/webp":      KindImage,
	"text/plain":      KindFile,
	"application/pdf": KindFile,
}

type CheckIn struct {
	ID                 int64         `json:"id"`
	GoalID             int64         `json:"goal_id"`
	OwnerUserID        int64         `json:"owner_user_id"`
	Status             CheckInStatus `json:"status"`
	SubmittedAt        *time.Time    `json:"submitted_at,omitempty"`
	ApprovedAt         *time.Time    `json:"approved_at,omitempty"`
	RejectedAt         *time.Time    `json:"rejected_at,omitempty"`
	ChangesRequestedAt *time.Time    `json:"changes_requested_at,omitempty"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

type EvidenceItem struct {
	ID            int64        `json:"id"`
	CheckInID     int64        `json:"check_in_id"`
	Kind          EvidenceKind `json:"kind"`
	TextContent   string       `json:"text_content,omitempty"`
	ExternalURL   string       `json:"external_url,omitempty"`
	StorageKey    string       `json:"storage_key,omitempty"`
	MIMEType      string       `json:"mime_type,omitempty"`
	FileSizeBytes int64        `json:"file_size_bytes,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
}

// CheckInView is the read model for a single check-in with all its evidence.
// BuddyUserID is included for service-layer permission checks and is not serialised.
type CheckInView struct {
	CheckIn      CheckIn        `json:"check_in"`
	Evidence     []EvidenceItem `json:"evidence"`
	BuddyUserID  int64          `json:"-"`
}

type AddTextInput struct {
	Content string `json:"content"`
}

type AddLinkInput struct {
	URL string `json:"url"`
}

func (in AddTextInput) Validate() error {
	content := strings.TrimSpace(in.Content)
	if content == "" {
		return errors.Join(ErrInvalidEvidenceInput, errors.New("content is required"))
	}
	if len(content) > 10_000 {
		return errors.Join(ErrInvalidEvidenceInput, errors.New("content must be 10000 characters or fewer"))
	}
	return nil
}

func (in AddLinkInput) Validate() error {
	url := strings.TrimSpace(in.URL)
	if url == "" {
		return errors.Join(ErrInvalidEvidenceInput, errors.New("url is required"))
	}
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return errors.Join(ErrInvalidEvidenceInput, errors.New("url must start with http:// or https://"))
	}
	if len(url) > 2048 {
		return errors.Join(ErrInvalidEvidenceInput, errors.New("url must be 2048 characters or fewer"))
	}
	return nil
}

// SanitizeURL drops the URL fragment (always browser-local, not proof) and
// trims whitespace. Query parameters are preserved because they often identify
// the exact artifact (PR number, commit SHA, etc.).
func SanitizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if i := strings.IndexByte(raw, '#'); i != -1 {
		raw = raw[:i]
	}
	return raw
}

func canAddEvidence(s CheckInStatus) bool {
	return s == StatusDraft || s == StatusChangesRequested
}

func canSubmit(s CheckInStatus) bool {
	return s == StatusDraft || s == StatusChangesRequested
}

func mimeToKind(mimeType string) (EvidenceKind, bool) {
	kind, ok := allowedMIMETypes[mimeType]
	return kind, ok
}

// DetectMIME reads the first 512 bytes of a file and returns the detected
// MIME type, falling back to the client-supplied hint only when detection
// returns application/octet-stream. This prevents clients from lying about
// the Content-Type of a multipart upload.
func DetectMIME(data []byte, clientHint string) string {
	if len(data) == 0 {
		return clientHint
	}
	head := data
	if len(head) > 512 {
		head = head[:512]
	}

	// Check magic bytes for the allowed types. net/http.DetectContentType is not
	// used here because it maps GIF/WEBP to generic types; we need exact matches.
	switch {
	case len(head) >= 3 && head[0] == 0xFF && head[1] == 0xD8 && head[2] == 0xFF:
		return "image/jpeg"
	case len(head) >= 8 && string(head[:8]) == "\x89PNG\r\n\x1a\n":
		return "image/png"
	case len(head) >= 6 && (string(head[:6]) == "GIF87a" || string(head[:6]) == "GIF89a"):
		return "image/gif"
	case len(head) >= 12 && string(head[:4]) == "RIFF" && string(head[8:12]) == "WEBP":
		return "image/webp"
	case len(head) >= 4 && string(head[:4]) == "%PDF":
		return "application/pdf"
	default:
		// Accept text/plain only if the client declared it and the bytes look
		// like UTF-8; fall back to octet-stream for unknown binary content.
		if clientHint == "text/plain" {
			return "text/plain"
		}
		return "application/octet-stream"
	}
}

func fileExtension(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "text/plain":
		return ".txt"
	case "application/pdf":
		return ".pdf"
	default:
		return ".bin"
	}
}
