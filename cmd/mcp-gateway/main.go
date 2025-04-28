package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/cnst"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"github.com/mcp-ecosystem/mcp-gateway/internal/core"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/storage"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/storage/helper"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/storage/notifier"
	pidHelper "github.com/mcp-ecosystem/mcp-gateway/pkg/helper"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/utils"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/version"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	configPath string
	pidFile    string
	serverLock sync.RWMutex
	httpServer *http.Server

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
	rootCmd = &cobra.Command{
		Use:   "mcp-gateway",
		Short: "MCP Gateway service",
		Long:  `MCP Gateway is a service that provides API gateway functionality for MCP ecosystem`,
		Run: func(cmd *cobra.Command, args []string) {
			run()
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "conf", cnst.MCPGatewayYaml, "path to configuration file, like /etc/mcp-gateway/apiserver.yaml")
	rootCmd.PersistentFlags().StringVar(&pidFile, "pid", "", "path to PID file")
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(reloadCmd)
}

func handleReload(ctx context.Context, logger *zap.Logger, store storage.Store, srv *core.Server, cfg *config.MCPGatewayConfig) {
	logger.Info("Reloading MCP configuration")

	mcpConfigs, err := store.List(ctx)
	if err != nil {
		logger.Fatal("Failed to load MCP configurations",
			zap.Error(err))
	}
	newMCPCfg, err := helper.MergeConfigs(mcpConfigs)
	if err != nil {
		logger.Fatal("failed to merge MCP configurations",
			zap.Error(err))
	}

	serverLock.Lock()
	defer serverLock.Unlock()

	newRouter := gin.New()

	if err := srv.RegisterRoutes(newRouter, newMCPCfg); err != nil {
		logger.Error("failed to register new routes",
			zap.Error(err))
		return
	}

	if err := srv.UpdateConfig(newMCPCfg); err != nil {
		logger.Error("failed to update server configuration",
			zap.Error(err))
		return
	}

	newServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: newRouter,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("failed to shutdown old server",
			zap.Error(err))
		return
	}

	httpServer = newServer
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("failed to start new server",
				zap.Error(err))
		}
	}()
}

func run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	cfg, cfgPath, err := config.LoadConfig[config.MCPGatewayConfig](cnst.MCPGatewayYaml)
	if err != nil {
		logger.Fatal("Failed to load service configuration",
			zap.String("path", cfgPath),
			zap.Error(err))
	}
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
	mcpCfg, err := helper.MergeConfigs(mcpConfigs)
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
	if err := srv.RegisterRoutes(router, mcpCfg); err != nil {
		logger.Fatal("failed to register routes",
			zap.Error(err))
	}

	httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
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
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("failed to start main server",
				zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-quit:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := httpServer.Shutdown(ctx); err != nil {
				cancel()
				logger.Error("failed to shutdown main server",
					zap.Error(err))
			}
			cancel()
			return
		case <-updateCh:
			logger.Info("Received reload signal")
			handleReload(ctx, logger, store, srv, cfg)
		}
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
