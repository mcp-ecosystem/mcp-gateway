# Gateway Configuration Editor

This component provides an intuitive interface for editing gateway configurations with two modes:

1. **YAML Mode**: Direct editing of the YAML configuration
2. **Form Mode**: User-friendly form interface for editing configuration

## Features

The form mode provides the following features:

- Basic configuration (name, tenant)
- Proxy type selection (HTTP or MCP)
- Router configuration with prefix and server mapping
- HTTP server configuration with:
  - Server details 
  - Allowed tools settings
  - Tool configurations
- MCP server configuration with:
  - Server type selection (stdio, sse, streamable-http)
  - Type-specific settings:
    - stdio: command, arguments, environment variables
    - sse/streamable-http: URL settings

## Implementation

The editor works by:
1. Parsing the YAML configuration
2. Providing a form interface to edit it
3. Converting form changes back to YAML on-the-fly
4. Preserving all original configuration values

## Usage

The ConfigEditor component takes the following props:

```typescript
interface ConfigEditorProps {
  config: string;          // YAML configuration as string
  onChange: (newConfig: string) => void;  // Called when config changes
  isDark: boolean;         // Dark mode flag
  editorOptions: any;      // Monaco editor options
}
```

## Examples

Two sample configurations are provided for testing:
- `test-config.yaml`: HTTP proxy configuration
- `test-mcp-config.yaml`: MCP proxy configuration 