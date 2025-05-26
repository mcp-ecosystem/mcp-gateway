export interface Gateway {
  name: string;
  config: string;
  
  id?: string;
  description?: string;
  repository?: {
    url: string;
    source: string;
    id: string;
  };
  version?: string;
  isPublished?: boolean;
  
  parsedConfig?: {
    tenant?: string;
    routers: Array<{
      server: string;
      prefix: string;
    }>;
    servers: Array<{
      name: string;
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

export interface RegistryServer {
  id: string;
  name: string;
  description: string;
  repository?: {
    url: string;
    source: string;
    id: string;
  };
  version_detail: {
    version: string;
    release_date?: string;
    is_latest: boolean;
  };
}

export interface PaginatedServers {
  servers: RegistryServer[];
  next?: string;
  total_count: number;
}
