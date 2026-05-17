package ai

type IntroRecommendation struct {
	CompanyName      string   `json:"company_name"`
	RoleTitle        string   `json:"role_title"`
	RecommendedAngle string   `json:"recommended_angle"`
	FitScore         int      `json:"fit_score"`
	FitSummary       string   `json:"fit_summary"`
	PrimaryMessage   string   `json:"primary_message"`
	SecondaryMessage string   `json:"secondary_message"`
	EvidenceUsed     []string `json:"evidence_used"`
	Cautions         []string `json:"cautions"`
}
