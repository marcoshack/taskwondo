package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/marcoshack/taskwondo/internal/model"
)

func TestValidateWorkItemTitle_TooLong(t *testing.T) {
	err := validateWorkItemTitle(strings.Repeat("x", model.MaxWorkItemTitleLen+1))
	if err == nil {
		t.Fatal("expected error for oversize title")
	}
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestValidateWorkItemTitle_WithinLimit(t *testing.T) {
	if err := validateWorkItemTitle(strings.Repeat("x", model.MaxWorkItemTitleLen)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWorkItemDescription_Nil(t *testing.T) {
	if err := validateWorkItemDescription(nil); err != nil {
		t.Fatalf("expected no error for nil description, got %v", err)
	}
}

func TestValidateWorkItemDescription_TooLong(t *testing.T) {
	big := strings.Repeat("x", model.MaxWorkItemDescriptionLen+1)
	if err := validateWorkItemDescription(&big); err == nil {
		t.Fatal("expected error for oversize description")
	}
}

func TestValidateWorkItemLabels_TooMany(t *testing.T) {
	labels := make([]string, model.MaxWorkItemLabels+1)
	for i := range labels {
		labels[i] = "l"
	}
	if err := validateWorkItemLabels(labels); err == nil {
		t.Fatal("expected error for too many labels")
	}
}

func TestValidateWorkItemLabels_LabelTooLong(t *testing.T) {
	labels := []string{strings.Repeat("x", model.MaxWorkItemLabelLen+1)}
	if err := validateWorkItemLabels(labels); err == nil {
		t.Fatal("expected error for oversize label")
	}
}

func TestValidateWorkItemCustomFields_TooMany(t *testing.T) {
	cf := make(map[string]interface{}, model.MaxWorkItemCustomFields+1)
	for i := 0; i <= model.MaxWorkItemCustomFields; i++ {
		cf[string(rune('a'+i%26))+strings.Repeat("0", i/26+1)] = "v"
	}
	if len(cf) <= model.MaxWorkItemCustomFields {
		t.Skipf("map builder underflowed: got %d keys", len(cf))
	}
	if err := validateWorkItemCustomFields(cf); err == nil {
		t.Fatal("expected error for too many custom fields")
	}
}

func TestValidateWorkItemCustomFields_KeyTooLong(t *testing.T) {
	cf := map[string]interface{}{strings.Repeat("k", model.MaxWorkItemCustomFieldKey+1): "v"}
	if err := validateWorkItemCustomFields(cf); err == nil {
		t.Fatal("expected error for oversize custom field key")
	}
}
