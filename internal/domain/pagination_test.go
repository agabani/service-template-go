package domain_test

import (
	"testing"

	"github.com/agabani/service-template-go/internal/domain"
)

func TestPageSize_defaults(t *testing.T) {
	if domain.PageSizeDefault <= 0 {
		t.Error("PageSizeDefault must be positive")
	}
	if domain.PageSizeMax < domain.PageSizeDefault {
		t.Error("PageSizeMax must be >= PageSizeDefault")
	}
}
