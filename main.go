package main

import (
	"context"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"

	"maas/services"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v4/pgxpool"
	"golang.org/x/exp/rand"
)

type Meme struct {
	ID    int     `json:"id"`
	Title string  `json:"title"`
	URL   string  `json:"url"`
	Query string  `json:"query"`
	Lat   float64 `json:"lat,omitempty"`
	Lon   float64 `json:"lon,omitempty"`
}

var tokenService *services.TokenService

func main() {
	// Initialize database connection
	dbpool, err := pgxpool.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbpool.Close()

	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	defer rdb.Close()

	// Initialize token service
	tokenService = services.NewTokenService(dbpool, rdb)

	app := fiber.New()

	app.Get("/memes", authMiddleware, getMemeHandler)
	app.Post("/tokens/add", addTokensHandler)
	app.Get("/tokens/balance", getBalanceHandler)

	app.Listen(":8080")
}

func authMiddleware(c fiber.Ctx) error {
	authToken := c.Get("Authorization")
	if authToken == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing authorization token",
		})
	}

	// Attempt to deduct tokens
	err := tokenService.DeductTokens(c.Context(), authToken)
	if err != nil {
		switch err {
		case services.ErrInsufficientTokens:
			return c.Status(http.StatusPaymentRequired).JSON(fiber.Map{
				"error": "Insufficient tokens",
			})
		case services.ErrInvalidToken:
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization token",
			})
		default:
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
	}

	return c.Next()
}

func addTokensHandler(c fiber.Ctx) error {
	var req struct {
		AuthToken string `json:"auth_token"`
		Amount    int    `json:"amount"`
	}

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	err := tokenService.AddTokens(c.Context(), req.AuthToken, req.Amount)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add tokens",
		})
	}

	return c.SendStatus(http.StatusOK)
}

func getBalanceHandler(c fiber.Ctx) error {
	authToken := c.Get("Authorization")
	if authToken == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing authorization token",
		})
	}

	balance, err := tokenService.GetBalance(c.Context(), authToken)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get balance",
		})
	}

	return c.JSON(fiber.Map{
		"balance": balance,
	})
}

func getMemeHandler(c fiber.Ctx) error {
	// Extract query parameters
	query := c.Query("query", "")
	lat := c.Query("lat", "")
	lon := c.Query("lon", "")

	var latitude, longitude float64
	var err error

	if lat != "" {
		latitude, err = strconv.ParseFloat(lat, 64)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid latitude value",
			})
		}
	}

	if lon != "" {
		longitude, err = strconv.ParseFloat(lon, 64)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid longitude value",
			})
		}
	}

	// Generate a random meme
	meme := generateMeme(query, latitude, longitude)

	return c.JSON(meme)
}

func generateMeme(query string, lat, lon float64) Meme {
	id := rand.Intn(1000)
	memes := []struct {
		title   string
		url     string
		memeLat float64
		memeLon float64
	}{
		{
			title:   "When you realize it's Monday again...",
			url:     "https://example.com/meme1.jpg",
			memeLat: 40.7128, // New York
			memeLon: -74.0060,
		},
		{
			title:   "Coding at 2am be like...",
			url:     "https://example.com/meme2.jpg",
			memeLat: 51.5074, // London
			memeLon: -0.1278,
		},
		{
			title:   "How it feels to fix a bug",
			url:     "https://example.com/meme3.jpg",
			memeLat: 35.6762, // Tokyo
			memeLon: 139.6503,
		},
		{
			title:   "That moment when your code compiles on the first try",
			url:     "https://example.com/meme4.jpg",
			memeLat: -33.8688, // Sydney
			memeLon: 151.2093,
		},
	}

	// Filter memes based on query and location
	var filteredMemes []struct {
		title   string
		url     string
		memeLat float64
		memeLon float64
	}

	const maxDistance = 5000.0 // Maximum distance in kilometers

	for _, meme := range memes {
		matchesQuery := query == "" || strings.Contains(strings.ToLower(meme.title), strings.ToLower(query))
		matchesLocation := (lat == 0 && lon == 0) ||
			(lat != 0 && lon != 0 && calculateDistance(lat, lon, meme.memeLat, meme.memeLon) <= maxDistance)

		if matchesQuery && matchesLocation {
			filteredMemes = append(filteredMemes, meme)
		}
	}

	// Return empty Meme if no matches found
	if len(filteredMemes) == 0 {
		return Meme{
			Query: query,
			Lat:   lat,
			Lon:   lon,
		}
	}

	// Select random meme from filtered results
	randomMeme := filteredMemes[rand.Intn(len(filteredMemes))]

	return Meme{
		ID:    id,
		Title: randomMeme.title,
		URL:   randomMeme.url,
		Query: query,
		Lat:   lat,
		Lon:   lon,
	}
}

// calculateDistance returns the distance between two points in kilometers using the Haversine formula
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0 // Earth's radius in kilometers

	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	// Haversine formula
	dlat := lat2Rad - lat1Rad
	dlon := lon2Rad - lon1Rad
	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distance := earthRadius * c

	return distance
}
