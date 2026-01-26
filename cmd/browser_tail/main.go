// browser_tail captures Chrome browser activity to structured JSONL logs.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/ajsharma/browser_tail/internal/cdp"
	"github.com/ajsharma/browser_tail/internal/config"
	"github.com/ajsharma/browser_tail/internal/logger"
)

var cfg = config.DefaultConfig()

var rootCmd = &cobra.Command{
	Use:   "browser_tail",
	Short: "Capture Chrome browser activity to structured JSONL logs",
	Long: `browser_tail connects to Chrome via the DevTools Protocol and captures
browser activity (navigation, network requests, console messages, errors)
to structured JSONL log files organized by site and tab.

Example:
  # Connect to existing Chrome (must be started with --remote-debugging-port=9222)
  browser_tail

  # Auto-launch Chrome with debugging enabled
  browser_tail --launch

  # Specify custom port and output directory
  browser_tail --port 9223 --output ./my_logs`,
	RunE: run,
}

func init() {
	// Connection flags
	rootCmd.Flags().StringVarP(&cfg.ChromePort, "port", "p", cfg.ChromePort,
		"Chrome remote debugging port")
	rootCmd.Flags().BoolVar(&cfg.AutoLaunch, "launch", cfg.AutoLaunch,
		"Auto-launch Chrome with debugging enabled")

	// Output flags
	rootCmd.Flags().StringVarP(&cfg.OutputDir, "output", "o", cfg.OutputDir,
		"Output directory for log files")
	rootCmd.Flags().DurationVar(&cfg.FlushInterval, "flush-interval", cfg.FlushInterval,
		"Flush interval for log buffering")
	rootCmd.Flags().IntVar(&cfg.BufferSize, "buffer-size", cfg.BufferSize,
		"Buffer size per tab in bytes")

	// Privacy flags
	rootCmd.Flags().BoolVarP(&cfg.Redact, "redact", "r", cfg.Redact,
		"Enable header redaction")
	rootCmd.Flags().BoolVar(&cfg.CaptureBodies, "capture-bodies", cfg.CaptureBodies,
		"Capture request/response bodies")
	rootCmd.Flags().IntVar(&cfg.BodySizeLimitKB, "body-size-limit", cfg.BodySizeLimitKB,
		"Max body size to capture in KB")

	// Event filtering flags
	rootCmd.Flags().BoolVar(&cfg.EnableNetwork, "network", cfg.EnableNetwork,
		"Enable network events")
	rootCmd.Flags().BoolVar(&cfg.EnableConsole, "console", cfg.EnableConsole,
		"Enable console events")
	rootCmd.Flags().BoolVar(&cfg.EnableErrors, "errors", cfg.EnableErrors,
		"Enable error events")
	rootCmd.Flags().BoolVar(&cfg.EnablePage, "page", cfg.EnablePage,
		"Enable page events")

	// Add --no-* flags for disabling
	rootCmd.Flags().Bool("no-network", false, "Disable network events")
	rootCmd.Flags().Bool("no-console", false, "Disable console events")
	rootCmd.Flags().Bool("no-errors", false, "Disable error events")
	rootCmd.Flags().Bool("no-page", false, "Disable page events")
	rootCmd.Flags().Bool("no-redact", false, "Disable redaction")

	// Version flag
	rootCmd.Version = config.Version
}

func run(cmd *cobra.Command, args []string) error {
	// Handle --no-* flags
	if noNetwork, _ := cmd.Flags().GetBool("no-network"); noNetwork {
		cfg.EnableNetwork = false
	}
	if noConsole, _ := cmd.Flags().GetBool("no-console"); noConsole {
		cfg.EnableConsole = false
	}
	if noErrors, _ := cmd.Flags().GetBool("no-errors"); noErrors {
		cfg.EnableErrors = false
	}
	if noPage, _ := cmd.Flags().GetBool("no-page"); noPage {
		cfg.EnablePage = false
	}
	if noRedact, _ := cmd.Flags().GetBool("no-redact"); noRedact {
		cfg.Redact = false
	}

	// Create output directory
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create file manager
	fm := logger.NewFileManager(cfg.OutputDir)
	fm.SetFlushInterval(cfg.FlushInterval)
	fm.SetBufferSize(cfg.BufferSize)

	// Create CDP manager
	manager := cdp.NewManager(cfg, fm)

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("\nReceived shutdown signal...")
		cancel()
	}()

	// Print startup info
	log.Printf("browser_tail %s", config.Version)
	log.Printf("Output directory: %s", cfg.OutputDir)
	log.Printf("Chrome port: %s", cfg.ChromePort)
	if cfg.AutoLaunch {
		log.Println("Auto-launching Chrome...")
	} else {
		log.Println("Connecting to existing Chrome...")
	}

	// Start monitoring
	errCh := make(chan error, 1)
	go func() {
		errCh <- manager.Start(ctx)
	}()

	// Wait for completion or error
	select {
	case err := <-errCh:
		if err != nil && ctx.Err() == nil {
			return err
		}
	case <-ctx.Done():
		// Give manager time to shut down gracefully
		time.Sleep(100 * time.Millisecond)
	}

	manager.Stop()
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
