package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientValidateBearer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uuid":        "550e8400-e29b-41d4-a716-446655440000",
			"username":    "a@b.c",
			"system_name": "sys",
			"is_admin":    true,
			"token":       map[string]any{"issued_at": "2020-01-01T00:00:00Z", "expires_at": "2030-01-01T00:00:00Z"},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.URL, 0)
	me, tt, err := c.ValidateBearer(context.Background(), "tok")
	require.NoError(t, err)
	assert.Equal(t, "a@b.c", me.Username)
	assert.Equal(t, "2020-01-01T00:00:00Z", tt.IssuedAt)
}

func TestClientValidateBearerEmptyHost(t *testing.T) {
	c := NewClient("", 0)
	_, _, err := c.ValidateBearer(context.Background(), "x")
	assert.Error(t, err)
}
