import { Gateway } from '@/types/gateway';

// Default configuration object for new or empty configurations
export const defaultConfig: Gateway = {
  name: '',
  tenant: 'default',
  routers: [],
  servers: [],
  tools: [],
  prompts: [],
  mcpServers: [],
  createdAt: new Date().toISOString(),
  updatedAt: new Date().toISOString()
};
