export interface Tenant {
  id: number;
  name: string;
  prefix: string;
  description: string;
  isActive: boolean;
}

export interface ConfigEditorProps {
  config: string;
  onChange: (newConfig: string) => void;
  isDark: boolean;
  editorOptions: Record<string, unknown>;
  isEditing?: boolean;
}

export interface GatewayConfig {
  name: string;
  tenant: string;
  routers?: Array<{
    server: string;
    prefix: string;
    cors?: Record<string, unknown>;
  }>;
  servers?: Array<{
    name: string;
    description: string;
    allowedTools: string[];
    config?: Record<string, unknown>;
  }>;
  tools?: Array<{
    name: string;
    description: string;
    method: string;
    endpoint: string;
    args?: Array<{
      name: string;
      position: string;
      required: boolean;
      type: string;
      description: string;
      default: string;
    }>;
    requestBody?: string;
    responseBody?: string;
    headers?: Record<string, string>;
    headersOrder?: string[];
  }>;
  mcpServers?: Array<{
    type: string;
    name: string;
    command?: string;
    args?: string[];
    env?: Record<string, string>;
    url?: string;
  }>;
}

export interface CorsConfig {
  allowOrigins?: string[];
  allowMethods?: string[];
  allowHeaders?: string[];
  exposeHeaders?: string[];
  allowCredentials?: boolean;
}

export interface KeyValueItem {
  key: string;
  value: string;
  description?: string;
}

export interface HeadersFormState {
  [toolIndex: number]: KeyValueItem[];
}

export interface EnvFormState {
  [serverIndex: number]: KeyValueItem[];
} 