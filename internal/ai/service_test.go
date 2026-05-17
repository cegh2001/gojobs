package ai

import "testing"

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
