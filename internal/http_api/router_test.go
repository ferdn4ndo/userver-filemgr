package http_api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRandomSig(t *testing.T) {
	s := randomSig()
	assert.GreaterOrEqual(t, len(s), 32)
	assert.LessOrEqual(t, len(s), 64)
}

func TestPaginate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/?limit=10&offset=5", nil)
	l, o := paginate(c)
	assert.Equal(t, 10, l)
	assert.Equal(t, 5, o)
}

func TestPageLinks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/storages/sid/media?limit=10&offset=0", nil)
	c.Request.URL.Path = "/storages/sid/media"

	links := pageLinks(c, 25, 10, 0)
	self := links["self"].(gin.H)["href"].(string)
	next := links["next"].(gin.H)["href"].(string)
	prev := links["previous"].(gin.H)["href"].(string)
	assert.Contains(t, self, "/storages/sid/media")
	assert.Equal(t, "/storages/sid/media?limit=10&offset=10", next)
	assert.Equal(t, "", prev)

	links2 := pageLinks(c, 5, 10, 0)
	assert.Equal(t, "", links2["next"].(gin.H)["href"].(string))
}
