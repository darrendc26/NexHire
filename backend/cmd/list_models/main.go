package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"google.golang.org/genai"
)

func main() {
	_ = godotenv.Load(".env")
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatalf("GEMINI_API_KEY is not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		log.Fatalf("Failed to create genai client: %v", err)
	}

	page, err := client.Models.List(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to list models: %v", err)
	}
	fmt.Println("Available Gemini Models:")
	for _, m := range page.Items {
		fmt.Printf("  • %s (DisplayName: %s)\n", m.Name, m.DisplayName)
	}
}
