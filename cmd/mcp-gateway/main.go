package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/cnst"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"github.com/mcp-ecosystem/mcp-gateway/internal/core"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/storage"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/storage/helper"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/storage/notifier"
	pidHelper "github.com/mcp-ecosystem/mcp-gateway/pkg/helper"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/logger"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/utils"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/version"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	configPath string
	pidFile    string

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number of mcp-gateway",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("mcp-gateway version %s\n", version.Get())
		},
	}
	reloadCmd = &cobra.Command{
		Use:   "reload",
		Short: "Reload the configuration of a running mcp-gateway instance",
		Run: func(cmd *cobra.Command, args []string) {
			// Load config to get PID path if not provided via command line
			cfg, _, err := config.LoadConfig[config.MCPGatewayConfig](configPath)
			if err != nil {
				fmt.Printf("Failed to load config: %v\n", err)
				os.Exit(1)
			}

			// Use PID from config if not provided via command line
			if pidFile == "" {
				pidFile = cfg.PID
			}

			if err := utils.SendSignalToPIDFile(pidHelper.GetPIDPath(pidFile), syscall.SIGHUP); err != nil {
				fmt.Printf("Failed to send reload signal: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Reload signal sent successfully")
		},
	}
	testCmd = &cobra.Command{
		Use:   "test",
		Short: "Test the configuration of mcp-gateway",
		Run: func(cmd *cobra.Command, args []string) {
			// Load config
			cfg, _, err := config.LoadConfig[config.MCPGatewayConfig](configPath)
			if err != nil {
				fmt.Printf("Failed to load config: %v\n", err)
				os.Exit(1)
			}

			// Initialize logger
			logger, err := logger.NewLogger(&cfg.Logger)
			if err != nil {
				fmt.Printf("Failed to initialize logger: %v\n", err)
				os.Exit(1)
			}
			defer logger.Sync()

			// Initialize storage
			store, err := storage.NewStore(logger, &cfg.Storage)
			if err != nil {
				fmt.Printf("Failed to initialize storage: %v\n", err)
				os.Exit(1)
			}

			// Load all MCP configurations
			mcpConfigs, err := store.List(context.Background())
			if err != nil {
				fmt.Printf("Failed to load MCP configurations: %v\n", err)
				os.Exit(1)
			}

			// Validate configurations
			if err := config.ValidateMCPConfigs(mcpConfigs); err != nil {
				var validationErr *config.ValidationError
				if errors.As(err, &validationErr) {
					fmt.Printf("Configuration validation failed: %v\n", validationErr)
				} else {
					fmt.Printf("Failed to validate configurations: %v\n", err)
				}
				os.Exit(1)
			}

			// Try to merge configurations
			if _, err := helper.MergeConfigs(mcpConfigs); err != nil {
				fmt.Printf("Failed to merge configurations: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("Configuration test is successful")
		},
	}
	rootCmd = &cobra.Command{
		Use:   cnst.CommandName,
		Short: "MCP Gateway service",
		Long:  `MCP Gateway is a service that provides API gateway functionality for MCP ecosystem`,
		Run: func(cmd *cobra.Command, args []string) {
			run()
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "conf", "c", cnst.MCPGatewayYaml, "path to configuration file, like /etc/mcp-gateway/mcp-gateway.yaml")
	rootCmd.PersistentFlags().StringVar(&pidFile, "pid", "", "path to PID file")
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(reloadCmd)
	rootCmd.AddCommand(testCmd)
}

func handleReload(ctx context.Context, logger *zap.Logger, store storage.Store, srv *core.Server, cfg *config.MCPGatewayConfig) {
	logger.Info("Reloading MCP configuration")
	//todo we can use hash or version to check if the configuration is changed
	mcpConfigs, err := store.List(ctx)
	if err != nil {
		logger.Error("Failed to load MCP configurations",
			zap.Error(err))
		return
	}

	// Validate configurations before merging
	if err := config.ValidateMCPConfigs(mcpConfigs); err != nil {
		var validationErr *config.ValidationError
		if errors.As(err, &validationErr) {
			logger.Error("Configuration validation failed",
				zap.String("error", validationErr.Error()))
		} else {
			logger.Error("failed to validate configurations",
				zap.Error(err))
		}
		return
	}

	newMCPCfgs, err := helper.MergeConfigs(mcpConfigs)
	if err != nil {
		logger.Error("failed to merge MCP configurations",
			zap.Error(err))
		return
	}

	// Update server configuration in place
	if err := srv.UpdateConfig(ctx, newMCPCfgs); err != nil {
		logger.Error("failed to update server configuration",
			zap.Error(err))
		return
	}

	logger.Info("Configuration reloaded successfully")
}

func handleMerge(ctx context.Context, logger *zap.Logger, srv *core.Server, mcpConfig *config.MCPConfig) {
	logger.Info("Merging MCP configuration")
	// Validate configurations before merging
	if err := config.ValidateMCPConfig(mcpConfig); err != nil {
		var validationErr *config.ValidationError
		if errors.As(err, &validationErr) {
			logger.Error("Configuration validation failed",
				zap.String("error", validationErr.Error()))
		} else {
			logger.Error("failed to validate configurations",
				zap.Error(err))
		}
		return
	}

	// Update server configuration in place
	if err := srv.MergeConfig(ctx, mcpConfig); err != nil {
		logger.Error("failed to merge server configuration",
			zap.Error(err))
		return
	}

	logger.Info("Configuration merged successfully", zap.Int("servers", len(mcpConfig.Servers)))
}
func run() {
	ctx, cancel := context.WithCancel(context.Background())

	// Load configuration first
	cfg, cfgPath, err := config.LoadConfig[config.MCPGatewayConfig](configPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to load service configuration: %v", err))
	}

	// Initialize logger with configuration
	logger, err := logger.NewLogger(&cfg.Logger)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	logger.Info("Loaded configuration", zap.String("path", cfgPath))

	// Initialize PID manager
	if pidFile == "" {
		pidFile = cfg.PID
	}

	pidManager := utils.NewPIDManagerFromConfig(pidFile)
	err = pidManager.WritePID()
	if err != nil {
		logger.Fatal("Failed to write PID file",
			zap.String("path", pidManager.GetPIDFile()),
			zap.Error(err))
	}
	defer pidManager.RemovePID()

	logger.Info("Starting mcp-gateway", zap.String("version", version.Get()))

	// Initialize storage and load initial configuration
	store, err := storage.NewStore(logger, &cfg.Storage)
	if err != nil {
		logger.Fatal("failed to initialize storage",
			zap.String("type", cfg.Storage.Type),
			zap.Error(err))
	}

	mcpConfigs, err := store.List(ctx)
	if err != nil {
		logger.Fatal("Failed to load MCP configurations",
			zap.Error(err))
	}

	// Validate configurations before merging
	if err := config.ValidateMCPConfigs(mcpConfigs); err != nil {
		var validationErr *config.ValidationError
		if errors.As(err, &validationErr) {
			logger.Fatal("Configuration validation failed",
				zap.String("error", validationErr.Error()))
		}
		logger.Fatal("failed to validate configurations",
			zap.Error(err))
	}

	mcpCfgs, err := helper.MergeConfigs(mcpConfigs)
	if err != nil {
		logger.Fatal("failed to merge MCP configurations",
			zap.Error(err))
	}

	srv, err := core.NewServer(logger, cfg)
	if err != nil {
		logger.Fatal("failed to create server",
			zap.Error(err))
	}

	router := gin.Default()
	// add health_check url for k8s readiness probe
	router.GET("/health_check", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Health check passed.",
		})
	})
	if err := srv.RegisterRoutes(ctx, router, mcpCfgs); err != nil {
		logger.Fatal("failed to register routes",
			zap.Error(err))
	}

	ntf, err := notifier.NewNotifier(ctx, logger, &cfg.Notifier)
	if err != nil {
		logger.Fatal("failed to initialize notifier",
			zap.Error(err))
	}
	updateCh, err := ntf.Watch(ctx)
	if err != nil {
		logger.Fatal("failed to start watching for updates",
			zap.Error(err))
	}

	go func() {
		logger.Info("Starting main server", zap.Int("port", cfg.Port))
		if err := router.Run(fmt.Sprintf(":%d", cfg.Port)); err != nil {
			logger.Fatal("failed to start main server",
				zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// periodically reload the configuration as a fallback mechanism for the notifier
	ticker := time.NewTicker(cfg.ReloadInterval)
	defer ticker.Stop()

	for {
		select {
		case <-quit:
			logger.Info("Received shutdown signal, starting graceful shutdown")

			// First cancel the main context to stop accepting new requests
			cancel()

			// Shutdown the MCP server to close all SSE connections
			err = srv.Shutdown(ctx)
			if err != nil {
				logger.Error("failed to shutdown MCP server",
					zap.Error(err))
			} else {
				logger.Info("MCP server shutdown completed successfully")
			}

			return
		case updateMCPConfig := <-updateCh:
			logger.Info("Received update signal")

			if updateMCPConfig == nil {
				logger.Warn("Updated configuration is nil, falling back to full reload")
				handleReload(ctx, logger, store, srv, cfg)
			} else {
				handleMerge(ctx, logger, srv, updateMCPConfig)
			}
		case <-ticker.C:
			logger.Info("Received ticker signal", zap.Bool("reload_switch", cfg.ReloadSwitch))
			if cfg.ReloadSwitch {
				handleReload(ctx, logger, store, srv, cfg)
			}
		}

	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
