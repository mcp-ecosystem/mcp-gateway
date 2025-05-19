package backend

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
	// Add new fields for testing
	Preferences struct {
		IsPublic  bool           `json:"isPublic"`
		ShowEmail bool           `json:"showEmail"`
		Theme     string         `json:"theme"`
		Tags      []string       `json:"tags"`
		Settings  map[string]any `json:"settings"`
	} `json:"preferences"`
}

var users = make(map[string]*User)

// HTTPServer implements the Server interface
type HTTPServer struct {
	server *http.Server
	router *gin.Engine
	logger *zap.Logger
}

func NewHTTPServer() *HTTPServer {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

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
		user.Preferences.Settings = make(map[string]any)

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
			IsPublic  bool           `json:"isPublic"`
			ShowEmail bool           `json:"showEmail"`
			Theme     string         `json:"theme"`
			Tags      []string       `json:"tags"`
			Settings  map[string]any `json:"settings"`
		}

		if err := c.ShouldBindJSON(&preferences); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user.Preferences = preferences
		c.JSON(http.StatusOK, user)
	})

	router.POST("/users/:email/avatar", func(c *gin.Context) {
		email := c.Param("email")
		_, exists := users[email]
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		avatarURL := c.PostForm("url")
		if avatarURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing url in form"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":   "avatar updated",
			"avatarUrl": avatarURL,
		})
	})

	return &HTTPServer{
		router: router,
		logger: logger,
	}
}

func (s *HTTPServer) Start(addr string) error {
	// Create server instance
	srv := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}
	s.server = srv

	go func() {
		s.logger.Info("Server is running on " + addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("Shutting down server...")
	return nil
}

func (s *HTTPServer) Stop() error {
	if s.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer s.logger.Sync()

	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Error("failed to shutdown server", zap.Error(err))
		return err
	}

	return nil
}
