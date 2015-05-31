package main

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)

func TestGenerateToken(t *testing.T) {
	credentials := &Credentials{"jriquelme", "secret"}
	key := []byte(strings.Repeat("s", 256))
	token, err := GenerateToken(credentials, 2*time.Hour, key)
	assert.Nil(t, err)
	t.Logf("token: %s", token)
}
