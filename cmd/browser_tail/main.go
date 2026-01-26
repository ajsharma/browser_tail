// browser_tail captures Chrome browser activity to structured JSONL logs.
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/ajsharma/browser_tail/internal/cdp"
	"github.com/ajsharma/browser_tail/internal/config"
	"github.com/ajsharma/browser_tail/internal/control"
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

	// Add control command
	rootCmd.AddCommand(controlCmd)
}

// Control command variables.
var (
	controlPort    string
	controlTimeout time.Duration
)

var controlCmd = &cobra.Command{
	Use:   "control",
	Short: "Control browser via CDP commands",
	Long: `Send commands to control the browser for automated testing.
Requires Chrome to be running with remote debugging enabled.

Example:
  browser_tail control navigate --url https://example.com
  browser_tail control click --selector "button#submit"
  browser_tail control type --selector "input[name=q]" --text "search query"
  browser_tail control eval --js "document.title"`,
}

var navigateCmd = &cobra.Command{
	Use:   "navigate",
	Short: "Navigate to a URL",
	RunE: func(cmd *cobra.Command, args []string) error {
		url, _ := cmd.Flags().GetString("url")
		if url == "" {
			return fmt.Errorf("--url is required")
		}

		ctrl, err := control.NewController(controlPort)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer ctrl.Close()
		ctrl.SetTimeout(controlTimeout)

		if err := ctrl.Navigate(url); err != nil {
			return fmt.Errorf("navigate failed: %w", err)
		}

		fmt.Printf("Navigated to: %s\n", url)
		return nil
	},
}

var clickCmd = &cobra.Command{
	Use:   "click",
	Short: "Click an element",
	RunE: func(cmd *cobra.Command, args []string) error {
		selector, _ := cmd.Flags().GetString("selector")
		if selector == "" {
			return fmt.Errorf("--selector is required")
		}

		ctrl, err := control.NewController(controlPort)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer ctrl.Close()
		ctrl.SetTimeout(controlTimeout)

		if err := ctrl.Click(selector); err != nil {
			return fmt.Errorf("click failed: %w", err)
		}

		fmt.Printf("Clicked: %s\n", selector)
		return nil
	},
}

var typeCmd = &cobra.Command{
	Use:   "type",
	Short: "Type text into an element",
	RunE: func(cmd *cobra.Command, args []string) error {
		selector, _ := cmd.Flags().GetString("selector")
		text, _ := cmd.Flags().GetString("text")
		if selector == "" {
			return fmt.Errorf("--selector is required")
		}
		if text == "" {
			return fmt.Errorf("--text is required")
		}

		ctrl, err := control.NewController(controlPort)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer ctrl.Close()
		ctrl.SetTimeout(controlTimeout)

		if err := ctrl.Type(selector, text); err != nil {
			return fmt.Errorf("type failed: %w", err)
		}

		fmt.Printf("Typed into %s: %s\n", selector, text)
		return nil
	},
}

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Evaluate JavaScript",
	RunE: func(cmd *cobra.Command, args []string) error {
		js, _ := cmd.Flags().GetString("js")
		if js == "" {
			return fmt.Errorf("--js is required")
		}

		ctrl, err := control.NewController(controlPort)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer ctrl.Close()
		ctrl.SetTimeout(controlTimeout)

		result, err := ctrl.Evaluate(js)
		if err != nil {
			return fmt.Errorf("eval failed: %w", err)
		}

		fmt.Println(result)
		return nil
	},
}

var screenshotCmd = &cobra.Command{
	Use:   "screenshot",
	Short: "Capture a screenshot",
	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")
		if output == "" {
			output = "screenshot.png"
		}

		ctrl, err := control.NewController(controlPort)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer ctrl.Close()
		ctrl.SetTimeout(controlTimeout)

		data, err := ctrl.Screenshot()
		if err != nil {
			return fmt.Errorf("screenshot failed: %w", err)
		}

		if output == "-" {
			// Output base64 to stdout
			fmt.Println(base64.StdEncoding.EncodeToString(data))
		} else {
			if err := os.WriteFile(output, data, 0o644); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}
			fmt.Printf("Screenshot saved to: %s\n", output)
		}

		return nil
	},
}

var titleCmd = &cobra.Command{
	Use:   "title",
	Short: "Get page title",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctrl, err := control.NewController(controlPort)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer ctrl.Close()
		ctrl.SetTimeout(controlTimeout)

		title, err := ctrl.GetTitle()
		if err != nil {
			return fmt.Errorf("failed to get title: %w", err)
		}

		fmt.Println(title)
		return nil
	},
}

var urlCmd = &cobra.Command{
	Use:   "url",
	Short: "Get current URL",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctrl, err := control.NewController(controlPort)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer ctrl.Close()
		ctrl.SetTimeout(controlTimeout)

		url, err := ctrl.GetURL()
		if err != nil {
			return fmt.Errorf("failed to get URL: %w", err)
		}

		fmt.Println(url)
		return nil
	},
}

var textCmd = &cobra.Command{
	Use:   "text",
	Short: "Get text content of an element",
	RunE: func(cmd *cobra.Command, args []string) error {
		selector, _ := cmd.Flags().GetString("selector")
		if selector == "" {
			return fmt.Errorf("--selector is required")
		}

		ctrl, err := control.NewController(controlPort)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer ctrl.Close()
		ctrl.SetTimeout(controlTimeout)

		text, err := ctrl.GetText(selector)
		if err != nil {
			return fmt.Errorf("failed to get text: %w", err)
		}

		fmt.Println(text)
		return nil
	},
}

func init() {
	// Control command flags
	controlCmd.PersistentFlags().StringVarP(&controlPort, "port", "p", "9222", "Chrome remote debugging port")
	controlCmd.PersistentFlags().DurationVarP(&controlTimeout, "timeout", "t", 30*time.Second, "Command timeout")

	// Navigate flags
	navigateCmd.Flags().String("url", "", "URL to navigate to")

	// Click flags
	clickCmd.Flags().String("selector", "", "CSS selector of element to click")

	// Type flags
	typeCmd.Flags().String("selector", "", "CSS selector of element")
	typeCmd.Flags().String("text", "", "Text to type")

	// Eval flags
	evalCmd.Flags().String("js", "", "JavaScript to evaluate")

	// Screenshot flags
	screenshotCmd.Flags().StringP("output", "o", "screenshot.png", "Output file (use - for base64 stdout)")

	// Text flags
	textCmd.Flags().String("selector", "", "CSS selector of element")

	// Add subcommands
	controlCmd.AddCommand(navigateCmd)
	controlCmd.AddCommand(clickCmd)
	controlCmd.AddCommand(typeCmd)
	controlCmd.AddCommand(evalCmd)
	controlCmd.AddCommand(screenshotCmd)
	controlCmd.AddCommand(titleCmd)
	controlCmd.AddCommand(urlCmd)
	controlCmd.AddCommand(textCmd)
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
