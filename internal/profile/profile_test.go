package profile

import (
	"strings"
	"testing"
)

func TestLoadAndDossier(t *testing.T) {
	candidateProfile, err := Load("../../profiles/carlos_gonzalez.json")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if candidateProfile.Name != "Carlos Eduardo Gonzalez Henriquez" {
		t.Fatalf("unexpected name: %q", candidateProfile.Name)
	}

	dossier := candidateProfile.Dossier()
	if !strings.Contains(dossier, "G-Aereo") {
		t.Fatalf("dossier should mention G-Aereo, got %q", dossier)
	}

	if !strings.Contains(dossier, "GoDojo") {
		t.Fatalf("dossier should mention GoDojo, got %q", dossier)
	}

	if !strings.Contains(dossier, "ve-commerce") {
		t.Fatalf("dossier should mention ve-commerce private repo evidence, got %q", dossier)
	}

	if !strings.Contains(dossier, "Computer Engineering") {
		t.Fatalf("dossier should mention education, got %q", dossier)
	}

	if !strings.Contains(dossier, "DevSecOps") {
		t.Fatalf("dossier should mention other skills, got %q", dossier)
	}
}
