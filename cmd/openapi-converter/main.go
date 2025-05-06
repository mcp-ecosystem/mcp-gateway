package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mcp-ecosystem/mcp-gateway/pkg/openapi"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	outputFormat string
	outputFile   string

	rootCmd = &cobra.Command{
		Use:   "openapi-converter [openapi-file]",
		Short: "Convert OpenAPI specification to MCP Gateway configuration",
		Long: `openapi-converter is a tool to convert OpenAPI specifications (JSON or YAML)
to MCP Gateway configuration format. It can read from a file or standard input
and output the result to a file or standard output.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create converter
			converter := openapi.NewConverter()

			// Read input file
			var input []byte
			var err error
			if args[0] == "-" {
				input, err = io.ReadAll(os.Stdin)
			} else {
				input, err = os.ReadFile(args[0])
			}
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}

			// Convert based on file extension
			var config interface{}
			ext := strings.ToLower(filepath.Ext(args[0]))
			switch ext {
			case ".json":
				config, err = converter.ConvertFromJSON(input)
			case ".yaml", ".yml":
				config, err = converter.ConvertFromYAML(input)
			default:
				// Try JSON first, then YAML
				config, err = converter.ConvertFromJSON(input)
				if err != nil {
					config, err = converter.ConvertFromYAML(input)
				}
			}
			if err != nil {
				return fmt.Errorf("failed to convert: %w", err)
			}

			// Marshal output
			var output []byte
			switch outputFormat {
			case "json":
				output, err = json.MarshalIndent(config, "", "  ")
			case "yaml":
				output, err = yaml.Marshal(config)
			default:
				return fmt.Errorf("unsupported output format: %s", outputFormat)
			}
			if err != nil {
				return fmt.Errorf("failed to marshal output: %w", err)
			}

			// Write output
			if outputFile == "" {
				fmt.Println(string(output))
			} else {
				if err := os.WriteFile(outputFile, output, 0644); err != nil {
					return fmt.Errorf("failed to write output: %w", err)
				}
			}

			return nil
		},
	}
)

func init() {
	rootCmd.Flags().StringVarP(&outputFormat, "format", "f", "yaml", "Output format (json or yaml)")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
