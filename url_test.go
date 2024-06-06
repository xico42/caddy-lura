package caddylura

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestApplyPlaceholders(t *testing.T) {
	subject := "/tenants/{{.Resp042_.http.request.header.X-Tenant-Id}}"
	expected := "/tenants/{http.request.header.X-Tenant-Id}"
	actual := applyCaddyPlaceholders(subject)

	assert.Equal(t, expected, actual)
}
