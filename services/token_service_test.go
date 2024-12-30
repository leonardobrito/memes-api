package services

import (
	"context"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestTokenService(t *testing.T) {
	// Setup test database and Redis connections
	db, err := pgxpool.Connect(context.Background(), "postgres://localhost:5432/test_db")
	assert.NoError(t, err)
	defer db.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	ts := NewTokenService(db, rdb)

	ctx := context.Background()
	testToken := "test_token"

	// Test adding tokens
	err = ts.AddTokens(ctx, testToken, 100)
	assert.NoError(t, err)

	// Test getting balance
	balance, err := ts.GetBalance(ctx, testToken)
	assert.NoError(t, err)
	assert.Equal(t, 100, balance)

	// Test deducting tokens
	err = ts.DeductTokens(ctx, testToken)
	assert.NoError(t, err)

	// Verify new balance
	balance, err = ts.GetBalance(ctx, testToken)
	assert.NoError(t, err)
	assert.Equal(t, 99, balance)

	// Test insufficient tokens
	for i := 0; i < 99; i++ {
		err = ts.DeductTokens(ctx, testToken)
		assert.NoError(t, err)
	}

	err = ts.DeductTokens(ctx, testToken)
	assert.Equal(t, ErrInsufficientTokens, err)
}
