package models

import (
	"time"
)

type Client struct {
	ID        int       `json:"id"`
	AuthToken string    `json:"auth_token"`
	CreatedAt time.Time `json:"created_at"`
}

type TokenTransaction struct {
	ID              int       `json:"id"`
	ClientID        int       `json:"client_id"`
	Amount          int       `json:"amount"`
	TransactionType string    `json:"transaction_type"`
	CreatedAt       time.Time `json:"created_at"`
}

type TokenBalance struct {
	ClientID int `json:"client_id"`
	Balance  int `json:"balance"`
}
