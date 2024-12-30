package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	TokenCost         = 1
	RedisKeyPrefix    = "token_balance:"
	RedisCacheTTL     = 24 * time.Hour
	CacheRefreshBatch = 100
)

var (
	ErrInsufficientTokens = errors.New("insufficient tokens")
	ErrInvalidToken       = errors.New("invalid auth token")
)

type TokenService struct {
	db          *pgxpool.Pool
	redis       *redis.Client
	cacheMutex  sync.RWMutex
	refreshChan chan string
}

func NewTokenService(db *pgxpool.Pool, redis *redis.Client) *TokenService {
	ts := &TokenService{
		db:          db,
		redis:       redis,
		refreshChan: make(chan string, CacheRefreshBatch),
	}
	go ts.backgroundCacheRefresh()
	return ts
}

// DeductTokens attempts to deduct tokens for an API call
func (ts *TokenService) DeductTokens(ctx context.Context, authToken string) error {
	// First, check the cache
	balance, err := ts.getBalanceFromCache(ctx, authToken)
	if err != nil {
		return err
	}

	if balance < TokenCost {
		return ErrInsufficientTokens
	}

	// Optimistically deduct tokens from cache
	newBalance := balance - TokenCost
	err = ts.updateBalanceInCache(ctx, authToken, newBalance)
	if err != nil {
		return err
	}

	// Record the transaction in the database asynchronously
	go func() {
		err := ts.recordTransaction(context.Background(), authToken, -TokenCost, "USAGE")
		if err != nil {
			// Log error and trigger cache refresh
			fmt.Printf("Error recording transaction: %v\n", err)
			ts.refreshChan <- authToken
		}
	}()

	return nil
}

// AddTokens adds tokens to a client's balance
func (ts *TokenService) AddTokens(ctx context.Context, authToken string, amount int) error {
	// Record the purchase in the database
	err := ts.recordTransaction(ctx, authToken, amount, "PURCHASE")
	if err != nil {
		return err
	}

	// Update the cache
	return ts.refreshBalance(ctx, authToken)
}

// GetBalance returns the current token balance for a client
func (ts *TokenService) GetBalance(ctx context.Context, authToken string) (int, error) {
	balance, err := ts.getBalanceFromCache(ctx, authToken)
	if err != nil {
		if err == redis.Nil {
			// Cache miss, refresh from database
			err = ts.refreshBalance(ctx, authToken)
			if err != nil {
				return 0, err
			}
			return ts.getBalanceFromCache(ctx, authToken)
		}
		return 0, err
	}
	return balance, nil
}

func (ts *TokenService) getBalanceFromCache(ctx context.Context, authToken string) (int, error) {
	ts.cacheMutex.RLock()
	defer ts.cacheMutex.RUnlock()

	balance, err := ts.redis.Get(ctx, RedisKeyPrefix+authToken).Int()
	if err != nil {
		return 0, err
	}
	return balance, nil
}

func (ts *TokenService) updateBalanceInCache(ctx context.Context, authToken string, balance int) error {
	ts.cacheMutex.Lock()
	defer ts.cacheMutex.Unlock()

	return ts.redis.Set(ctx, RedisKeyPrefix+authToken, balance, RedisCacheTTL).Err()
}

func (ts *TokenService) refreshBalance(ctx context.Context, authToken string) error {
	// Calculate balance from database
	var balance int
	err := ts.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN transaction_type = 'PURCHASE' THEN amount ELSE -amount END), 0)
		FROM token_transactions
		WHERE client_id = (SELECT id FROM clients WHERE auth_token = $1)
	`, authToken).Scan(&balance)
	if err != nil {
		return err
	}

	// Update cache
	return ts.updateBalanceInCache(ctx, authToken, balance)
}

func (ts *TokenService) recordTransaction(ctx context.Context, authToken string, amount int, transactionType string) error {
	_, err := ts.db.Exec(ctx, `
		INSERT INTO token_transactions (client_id, amount, transaction_type)
		SELECT id, $2, $3
		FROM clients
		WHERE auth_token = $1
	`, authToken, amount, transactionType)
	return err
}

func (ts *TokenService) backgroundCacheRefresh() {
	for authToken := range ts.refreshChan {
		ctx := context.Background()
		err := ts.refreshBalance(ctx, authToken)
		if err != nil {
			fmt.Printf("Error refreshing cache for token %s: %v\n", authToken, err)
		}
	}
}
