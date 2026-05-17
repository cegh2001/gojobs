package ai

import (
	"errors"
	"strings"
	"testing"
)

func TestDecodeRecommendationPayloadAcceptsFullFinalChunk(t *testing.T) {
	fullJSON := `{"company_name":"ClaimSorted","role_title":"AI Engineer","recommended_angle":"ai-agents","fit_score":84,"fit_summary":"Strong fit.","primary_message":"Primary","secondary_message":"Secondary","evidence_used":["RAG systems"],"cautions":["Do not mention relocation."]}`

	recommendation, err := decodeRecommendationPayload(fullJSON, fullJSON+fullJSON)
	if err != nil {
		t.Fatalf("decodeRecommendationPayload() error = %v", err)
	}

	if recommendation.CompanyName != "ClaimSorted" {
		t.Fatalf("CompanyName = %q, want %q", recommendation.CompanyName, "ClaimSorted")
	}
}

func TestDecodeRecommendationPayloadAcceptsAccumulatedChunks(t *testing.T) {
	finalChunk := `}`
	accumulated := `{"company_name":"ClaimSorted","role_title":"AI Engineer","recommended_angle":"ai-agents","fit_score":84,"fit_summary":"Strong fit.","primary_message":"Primary","secondary_message":"Secondary","evidence_used":["RAG systems"],"cautions":["Do not mention relocation."]}`

	recommendation, err := decodeRecommendationPayload(finalChunk, accumulated)
	if err != nil {
		t.Fatalf("decodeRecommendationPayload() error = %v", err)
	}

	if recommendation.RoleTitle != "AI Engineer" {
		t.Fatalf("RoleTitle = %q, want %q", recommendation.RoleTitle, "AI Engineer")
	}
}

func TestIsRetryableModelError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "internal 500", err: errors.New("Error 500, Status: INTERNAL"), want: true},
		{name: "unavailable 503", err: errors.New("Error 503, Status: UNAVAILABLE"), want: true},
		{name: "decode error", err: errors.New("decode response JSON: unexpected end of JSON input"), want: false},
	}

	for _, test := range tests {
		if got := isRetryableModelError(test.err); got != test.want {
			t.Fatalf("%s: got %v want %v", test.name, got, test.want)
		}
	}
}

func TestDecodeRecommendationPayloadReturnsHelpfulEmptyError(t *testing.T) {
	_, err := decodeRecommendationPayload("")
	if err == nil {
		t.Fatalf("expected an error for empty payload")
	}

	if !strings.Contains(err.Error(), "empty response") {
		t.Fatalf("expected empty response error, got %v", err)
	}
}
