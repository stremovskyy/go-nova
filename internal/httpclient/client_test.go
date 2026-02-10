package httpclient

import (
	"testing"

	"github.com/google/uuid"
)

func TestNextRequestIDIsUUIDv4(t *testing.T) {
	id := nextRequestID()

	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("request_id must be a valid UUID, got %q: %v", id, err)
	}
	if parsed.Version() != 4 {
		t.Fatalf("request_id must be UUID v4, got version %d (%q)", parsed.Version(), id)
	}
	if parsed.Variant() != uuid.RFC4122 {
		t.Fatalf("request_id must use RFC4122 variant, got %v (%q)", parsed.Variant(), id)
	}
}
