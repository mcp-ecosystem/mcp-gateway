export interface Gateway {
  name: string;
  tenant: string;
  config: string;
  parsedConfig?: {
    name: string;
    tenant?: string;
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
      policy?: 'onStart' | 'onDemand';
      preinstalled?: boolean;
    }>;
  };
}
