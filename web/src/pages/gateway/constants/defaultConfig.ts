import { GatewayConfig } from '../types';

// Default configuration object for new or empty configurations
export const defaultConfig: GatewayConfig = {
  name: "",
  tenant: "default",
  routers: [],
  servers: [],
  tools: [],
  mcpServers: []
};
