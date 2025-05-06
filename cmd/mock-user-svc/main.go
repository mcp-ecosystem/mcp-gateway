package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/version"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
	// Add new fields for testing
	Preferences struct {
		IsPublic  bool     `json:"isPublic"`
		ShowEmail bool     `json:"showEmail"`
		Theme     string   `json:"theme"`
		Tags      []string `json:"tags"`
	} `json:"preferences"`
}

var users = make(map[string]*User)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of mock-user-svc",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("mock-user-svc version %s\n", version.Get())
	},
}

var rootCmd = &cobra.Command{
	Use:   "mock-user-svc",
	Short: "Mock User Service",
	Long:  `Mock User Service provides mock user management functionality`,
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func run() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	logger.Info("Starting mock-user-svc", zap.String("version", version.Get()))

	// Initialize router
	router := gin.Default()

	// Register routes
	router.POST("/users", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Generate ID and timestamp
		user.ID = uuid.New().String()
		user.CreatedAt = time.Now()

		// Initialize default values
		user.Preferences.IsPublic = false
		user.Preferences.ShowEmail = true
		user.Preferences.Theme = "light"
		user.Preferences.Tags = []string{}

		// Store user
		users[user.Email] = &user

		c.JSON(http.StatusCreated, user)
	})

	router.GET("/users/email/:email", func(c *gin.Context) {
		email := c.Param("email")
		user, exists := users[email]
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		c.JSON(http.StatusOK, user)
	})

	// Add new endpoint for updating user preferences
	router.PUT("/users/:email/preferences", func(c *gin.Context) {
		email := c.Param("email")
		user, exists := users[email]
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		var preferences struct {
			IsPublic  bool     `json:"isPublic"`
			ShowEmail bool     `json:"showEmail"`
			Theme     string   `json:"theme"`
			Tags      []string `json:"tags"`
		}

		if err := c.ShouldBindJSON(&preferences); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user.Preferences = preferences
		c.JSON(http.StatusOK, user)
	})

	// Start server
	srv := &http.Server{
		Addr:    ":5236",
		Handler: router,
	}

	go func() {
		logger.Info("Server is running on :5236")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("failed to shutdown server", zap.Error(err))
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
