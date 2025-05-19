export interface Gateway {
  name: string;
  config: string;
  parsedConfig?: {
    routers: Array<{
      server: string;
      prefix: string;
    }>;
    servers: Array<{
      name: string;
      namespace: string;
      description: string;
      allowedTools: string[];
    }>;
    tools: Array<{
      name: string;
      description: string;
      method: string;
    }>;
    mcpServers?: Array<{
      type: string;
      name: string;
      command?: string;
      args?: string[];
      env?: Record<string, string>;
      url?: string;
    }>;
  };
}
