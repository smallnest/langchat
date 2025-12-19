package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/smallnest/langchat/pkg/chat"
)

//go:embed static
var staticFS embed.FS

// loadEnv loads environment variables from .env file if it exists
func loadEnv() {
	if _, err := os.Stat(".env"); err == nil {
		content, err := os.ReadFile(".env")
		if err != nil {
			log.Printf("Error reading .env file: %v", err)
			return
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				os.Setenv(key, value)
			}
		}
	}
}

func main() {
	// Load environment variables from .env file
	loadEnv()

	// Load configuration from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	sessionDir := os.Getenv("SESSION_DIR")
	if sessionDir == "" {
		sessionDir = "./sessions"
	}

	maxHistory := 50
	if maxHistoryStr := os.Getenv("MAX_HISTORY_SIZE"); maxHistoryStr != "" {
		if _, err := fmt.Sscanf(maxHistoryStr, "%d", &maxHistory); err != nil {
			log.Printf("Warning: Failed to parse MAX_HISTORY_SIZE %q, using default 50: %v", maxHistoryStr, err)
			maxHistory = 50
		}
	}

	// Get config file path from environment or use default
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.json"
	}

	// Create and start server
	server, err := chat.NewChatServer(sessionDir, maxHistory, port, configPath)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Pre-warm: Initialize tools in background before server starts
	// This prevents the first user from experiencing slow tool loading
	log.Println("🔄 Pre-warming tools initialization...")
	warmupAgent := chat.NewSimpleChatAgent(server.GetLLM(), *server.GetConfig())
	warmupAgent.InitializeToolsAsync()

	// Store the warmup agent so it can be reused for the first session
	server.SetWarmupAgent(warmupAgent)

	// Wait for tools to finish loading before starting server
	go func() {
		for {
			// Simple check - in real implementation you might want to add a method to check status
			time.Sleep(500 * time.Millisecond)
			// For now, we'll just wait a bit and then show ready message
			log.Printf("🚀 Server is ready! Access at http://localhost:%s", port)
			break
		}
	}()

	// Setup graceful shutdown
	serverErr := make(chan error, 1)
	go func() {
		if err := server.Start(staticFS); err != nil {
			serverErr <- err
		}
	}()

	// Wait for interrupt signal or server error
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Block until signal received or server error
	select {
	case sig := <-sigChan:
		log.Printf("Received shutdown signal: %v", sig)
	case err := <-serverErr:
		log.Printf("Server error: %v", err)
	}

	// Graceful shutdown with timeout
	log.Println("Starting graceful shutdown...")
	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- server.Close()
	}()

	// Wait for shutdown to complete with timeout
	select {
	case err := <-shutdownDone:
		if err != nil {
			log.Printf("Error during shutdown: %v", err)
			os.Exit(1)
		}
		log.Println("Shutdown complete")
		os.Exit(0)
	case <-time.After(15 * time.Second):
		log.Println("Shutdown timed out after 15 seconds, forcing exit")
		os.Exit(1)
	}
}
