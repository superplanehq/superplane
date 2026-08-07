package models

import "testing"

func TestValidateDomainType(t *testing.T) {
	if err := ValidateDomainType(DomainTypeOrganization); err != nil {
		t.Fatalf("expected org domain type to be valid, got: %v", err)
	}

	if err := ValidateDomainType(DomainTypeCanvas); err == nil {
		t.Fatal("expected canvas domain type to be rejected by ValidateDomainType (RBAC is org-only)")
	}

	if err := ValidateDomainType("bogus"); err == nil {
		t.Fatal("expected bogus domain type to be rejected")
	}
}

func TestValidateSecretDomainType(t *testing.T) {
	if err := ValidateSecretDomainType(DomainTypeOrganization); err != nil {
		t.Fatalf("expected org domain type to be valid, got: %v", err)
	}

	if err := ValidateSecretDomainType(DomainTypeCanvas); err != nil {
		t.Fatalf("expected canvas domain type to be valid, got: %v", err)
	}

	if err := ValidateSecretDomainType("bogus"); err == nil {
		t.Fatal("expected bogus domain type to be rejected")
	}
}
