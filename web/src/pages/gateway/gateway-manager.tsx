import { Card, CardBody, Button, Modal, ModalContent, ModalHeader, ModalBody, ModalFooter, useDisclosure, Chip, Dropdown, DropdownTrigger, DropdownMenu, DropdownItem, Autocomplete, AutocompleteItem } from "@heroui/react";
import { Icon } from '@iconify/react';
import Editor from '@monaco-editor/react';
import yaml from 'js-yaml';
import { configureMonacoYaml } from 'monaco-yaml';
import React from 'react';
import { useTranslation } from 'react-i18next';

import { getMCPServers, createMCPServer, updateMCPServer, deleteMCPServer, syncMCPServers, getUserAuthorizedTenants, getTenant } from '../../services/api';
import { toast } from '../../utils/toast';

import OpenAPIImport from './components/OpenAPIImport';

declare global {
  interface Window {
    monaco: {
      languages: {
        yaml: {
          yamlDefaults: {
            setDiagnosticsOptions: (options: { enableSchemaRequest: boolean; schemas: Array<{ uri: string; fileMatch: string[] }> }) => void;
          };
        };
      };
    };
  }
}

interface Gateway {
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

interface ServerConfig {
  name: string;
  namespace: string;
  description: string;
  allowedTools: string[];
}

interface RouterConfig {
  server: string;
  prefix: string;
}

interface ToolConfig {
  name: string;
  description: string;
  method: string;
}

interface Tenant {
  id: number;
  name: string;
  prefix: string;
  description: string;
  isActive: boolean;
}

export function GatewayManager() {
  const { t } = useTranslation();
  const {isOpen, onOpen, onOpenChange} = useDisclosure();
  const {isOpen: isCreateOpen, onOpen: onCreateOpen, onOpenChange: onCreateOpenChange} = useDisclosure();
  const {isOpen: isImportOpen, onOpen: onImportOpen, onOpenChange: onImportOpenChange} = useDisclosure();
  const [mcpservers, setMCPServers] = React.useState<Gateway[]>([]);
  const [currentMCPServer, setCurrentMCPServer] = React.useState<Gateway | null>(null);
  const [editConfig, setEditConfig] = React.useState('');
  const [newConfig, setNewConfig] = React.useState('');
  const [parsedMCPServers, setParsedMCPServers] = React.useState<Gateway[]>([]);
  const [isLoading, setIsLoading] = React.useState(true);
  const [tenants, setTenants] = React.useState<Tenant[]>([]);
  const [selectedTenants, setSelectedTenants] = React.useState<Tenant[]>([]);
  const [tenantInputValue, setTenantInputValue] = React.useState('');
  const [isDark, setIsDark] = React.useState(() => {
    return document.documentElement.classList.contains('dark');
  });

  // Listen for theme changes
  React.useEffect(() => {
    const observer = new globalThis.MutationObserver((mutations) => {
      mutations.forEach((mutation) => {
        if (mutation.attributeName === 'class') {
          setIsDark(document.documentElement.classList.contains('dark'));
        }
      });
    });

    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class']
    });

    return () => observer.disconnect();
  }, []);

  // Configure Monaco YAML
  React.useEffect(() => {
    const monaco = window.monaco;
    if (monaco) {
      configureMonacoYaml(monaco, {
        enableSchemaRequest: true,
        schemas: [
          {
            uri: '',
            fileMatch: ['*.yml', '*.yaml'],
          },
        ],
      });
    }
  }, []);

  // Get MCP servers list
  React.useEffect(() => {
    const fetchMCPServers = async () => {
      try {
        setIsLoading(true);
        // Use the first selected tenant ID for filtering if available
        const tenantId = selectedTenants.length > 0 ? selectedTenants[0].id : undefined;
        const servers = await getMCPServers(tenantId);
        setMCPServers(servers);
      } catch {
        toast.error(t('errors.fetch_mcp_servers'));
      } finally {
        setIsLoading(false);
      }
    };

    fetchMCPServers();
  }, [t, selectedTenants]);

  // Get user's authorized tenants
  React.useEffect(() => {
    const fetchAuthorizedTenants = async () => {
      try {
        const tenantsData = await getUserAuthorizedTenants();
        setTenants(tenantsData);
      } catch {
        toast.error(t('errors.fetch_authorized_tenants'));
      }
    };

    fetchAuthorizedTenants();
  }, [t]);

  const handleEdit = (server: Gateway) => {
    setCurrentMCPServer(server);
    setEditConfig(server.config);
    onOpen();
  };

  // Validate if router prefixes start with tenant prefix
  const validateRouterPrefixes = async (config: string): Promise<boolean> => {
    try {
      const parsedConfig = yaml.load(config) as { tenant: string, routers: Array<{ prefix: string }> };
      
      if (!parsedConfig.tenant || !parsedConfig.routers) {
        return true; // Skip validation if tenant or routers are missing
      }
      
      // Get tenant information
      const tenant = await getTenant(parsedConfig.tenant);
      if (!tenant) {
        return true; // Skip validation if tenant not found
      }
      
      // Normalize tenant prefix
      let tenantPrefix = tenant.prefix;
      if (!tenantPrefix.startsWith('/')) {
        tenantPrefix = '/' + tenantPrefix;
      }
      tenantPrefix = tenantPrefix.endsWith('/') ? tenantPrefix.slice(0, -1) : tenantPrefix;
      
      // Check if all router prefixes start with the tenant prefix
      for (const router of parsedConfig.routers) {
        // Normalize router prefix
        let routerPrefix = router.prefix;
        if (!routerPrefix.startsWith('/')) {
          routerPrefix = '/' + routerPrefix;
        }
        routerPrefix = routerPrefix.endsWith('/') ? routerPrefix.slice(0, -1) : routerPrefix;
        
        // Allow exact match
        if (routerPrefix === tenantPrefix) {
          continue;
        }
        
        // Router prefix must start with tenant prefix followed by a slash
        if (!routerPrefix.startsWith(tenantPrefix + '/')) {
          toast.error(t('errors.router_prefix_error'), {
            duration: 3000,
          });
          return false;
        }
      }
      
      return true;
    } catch {
      toast.error(t('errors.validate_router_prefix_failed'), {
        duration: 3000,
      });
      return false;
    }
  };

  const handleSave = async () => {
    try {
      // Validate YAML
      yaml.load(editConfig);
      
      // Validate router prefix
      const isValidPrefix = await validateRouterPrefixes(editConfig);
      if (!isValidPrefix) {
        return;
      }

      if (currentMCPServer) {
        await updateMCPServer(currentMCPServer.name, editConfig);
        const tenantId = selectedTenants.length > 0 ? selectedTenants[0].id : undefined;
        const servers = await getMCPServers(tenantId);
        setMCPServers(servers);
        toast.success(t('gateway.edit_success'));
      }
      onOpenChange();
    } catch {
      toast.error(t('gateway.edit_failed'));
    }
  };

  const handleDelete = async (server: Gateway) => {
    try {
      await deleteMCPServer(server.name);
      const tenantId = selectedTenants.length > 0 ? selectedTenants[0].id : undefined;
      const servers = await getMCPServers(tenantId);
      setMCPServers(servers);
      toast.success(t('gateway.delete_success'));
    } catch {
      toast.error(t('gateway.delete_failed'));
    }
  };

  const handleSync = async () => {
    try {
      setIsLoading(true);
      await syncMCPServers();
      const tenantId = selectedTenants.length > 0 ? selectedTenants[0].id : undefined;
      const servers = await getMCPServers(tenantId);
      setMCPServers(servers);
      toast.success(t('gateway.sync_success'));
    } catch {
      toast.error(t('gateway.sync_failed'));
    } finally {
      setIsLoading(false);
    }
  };

  const handleCopyToClipboard = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      toast.success(t('common.copied', { text }));
    } catch {
      toast.error(t('common.copy_failed'));
    }
  };

  const handleCreate = async () => {
    try {
      // Validate YAML format first
      try {
        yaml.load(newConfig);
      } catch {
        toast.error(t('errors.invalid_yaml'));
        return;
      }
      
      // Validate router prefix
      const isValidPrefix = await validateRouterPrefixes(newConfig);
      if (!isValidPrefix) {
        return;
      }

      // If YAML is valid, proceed with creation
      await createMCPServer(newConfig);
      const tenantId = selectedTenants.length > 0 ? selectedTenants[0].id : undefined;
      const servers = await getMCPServers(tenantId);
      setMCPServers(servers);
      onCreateOpenChange();
      setNewConfig('');
      toast.success(t('gateway.add_success'));
    } catch {
      toast.error(t('gateway.add_failed'));
    }
  };

  const handleImportSuccess = async () => {
    try {
      const tenantId = selectedTenants.length > 0 ? selectedTenants[0].id : undefined;
      const servers = await getMCPServers(tenantId);
      setMCPServers(servers);
      onImportOpenChange();
      toast.success(t('gateway.import_success'));
    } catch {
      toast.error(t('gateway.import_failed'));
    }
  };

  React.useEffect(() => {
    const parseConfigs = () => {
      const parsed = mcpservers.map(server => {
        try {
          const config = yaml.load(server.config) as Gateway['parsedConfig'];
          return { ...server, parsedConfig: config };
        } catch {
          toast.error(t('errors.parse_config', { name: server.name }));
          return server;
        }
      });
      setParsedMCPServers(parsed);
    };
    parseConfigs();
  }, [mcpservers, t]);

  const editorOptions = {
    minimap: { enabled: false },
    fontSize: 14,
    lineNumbers: 'on',
    roundedSelection: false,
    scrollBeyondLastLine: false,
    readOnly: false,
    automaticLayout: true,
    ...(isDark ? {
      'editor.background': '#1E2228',
      'editor.foreground': '#E5E7EB',
      'editor.lineHighlightBackground': '#23272E',
      'editor.selectionBackground': '#4C6BCF40',
      'editor.inactiveSelectionBackground': '#4C6BCF20',
      'editor.lineHighlightBorder': '#2D3238',
      'editorCursor.foreground': '#4C6BCF',
      'editorWhitespace.foreground': '#4B5563',
    } : {
      'editor.background': '#F8F9FA',
      'editor.foreground': '#0B0F1A',
      'editor.lineHighlightBackground': '#F1F3F5',
      'editor.selectionBackground': '#4C6BCF40',
      'editor.inactiveSelectionBackground': '#4C6BCF20',
      'editor.lineHighlightBorder': '#E5E7EB',
      'editorCursor.foreground': '#4C6BCF',
      'editorWhitespace.foreground': '#A6B6E8',
    })
  };

  // Define custom filter function for tenants
  const customTenantFilter = (inputValue: string, items: Tenant[]) => {
    const lowerCaseInput = inputValue.toLowerCase();
    return items.filter(item => 
      item.name.toLowerCase().includes(lowerCaseInput) || 
      item.prefix.toLowerCase().includes(lowerCaseInput)
    );
  };

  const handleTenantSelect = (key: React.Key | null) => {
    if (key === null) return;
    
    const tenant = tenants.find(t => t.id === parseInt(key.toString(), 10));
    if (tenant && !selectedTenants.some(t => t.id === tenant.id)) {
      setSelectedTenants(prev => [...prev, tenant]);
    }
    setTenantInputValue('');
  };
  
  const handleRemoveTenant = (tenantId: number) => {
    setSelectedTenants(prev => prev.filter(t => t.id !== tenantId));
  };
  
  // Filter out already selected tenants for selection
  const availableTenants = React.useMemo(() => {
    return tenants.filter(tenant => 
      !selectedTenants.some(selected => selected.id === tenant.id)
    );
  }, [tenants, selectedTenants]);

  return (
    <div className="container mx-auto p-4">
      <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold">{t('gateway.title')}</h1>
        <div className="flex gap-2">
          <Button
            color="primary"
            onPress={onCreateOpen}
            startContent={<Icon icon="material-symbols:add" />}
          >
            {t('gateway.add')}
          </Button>
          <Button
            color="secondary"
            variant="flat"
            onPress={onImportOpen}
            startContent={<Icon icon="material-symbols:upload" />}
            className="bg-purple-500 hover:bg-purple-600 text-white"
          >
            {t('gateway.import_openapi')}
          </Button>
          <Button
            color="default"
            onPress={handleSync}
            isLoading={isLoading}
            startContent={<Icon icon="material-symbols:sync" />}
          >
            {t('gateway.sync')}
          </Button>
        </div>
      </div>

      <div className="mb-4 max-w-md">
        <label className="block text-sm font-medium mb-1">{t('gateway.select_tenant')}</label>
        
        {/* Display selected tenants */}
        <div className="flex flex-wrap gap-1 mb-2">
          {selectedTenants.map(tenant => (
            <Chip 
              key={tenant.id} 
              onClose={() => handleRemoveTenant(tenant.id)}
              variant="flat"
            >
              {`${tenant.name}(${tenant.prefix})`}
            </Chip>
          ))}
        </div>
        
        <Autocomplete
          placeholder={t('gateway.search_tenant')}
          defaultItems={availableTenants}
          inputValue={tenantInputValue}
          onInputChange={setTenantInputValue}
          onSelectionChange={handleTenantSelect}
          menuTrigger="focus"
          isClearable
          startContent={<Icon icon="lucide:search" className="text-gray-400" />}
          listboxProps={{
            emptyContent: t('common.no_results')
          }}
          items={customTenantFilter(tenantInputValue, availableTenants)}
        >
          {(tenant) => (
            <AutocompleteItem 
              key={tenant.id.toString()} 
              textValue={`${tenant.name}(${tenant.prefix})`}
            >
              <div className="flex flex-col">
                <span>{tenant.name}</span>
                <span className="text-xs text-gray-500">{tenant.prefix}</span>
              </div>
            </AutocompleteItem>
          )}
        </Autocomplete>
      </div>

      {isLoading ? (
        <div className="flex justify-center items-center h-32">
          <Icon icon="lucide:loader-2" className="animate-spin text-2xl" />
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {(parsedMCPServers || []).map((server) => (
            <Card key={server.name} className="w-full hover:shadow-lg transition-shadow bg-card">
              <CardBody className="flex flex-col gap-3 p-4">
                <div className="flex justify-between items-center">
                  <h3 className="text-lg font-semibold truncate">{server.name}</h3>
                  <div className="flex gap-2">
                    <Button
                      isIconOnly
                      color="primary"
                      variant="light"
                      size="sm"
                      onPress={() => handleEdit(server)}
                      aria-label={t('gateway.edit')}
                    >
                      <Icon icon="lucide:edit" className="text-lg" />
                    </Button>
                    <Dropdown>
                      <DropdownTrigger>
                        <Button
                          isIconOnly
                          color="danger"
                          variant="light"
                          size="sm"
                          aria-label={t('common.actions')}
                        >
                          <Icon icon="lucide:more-vertical" className="text-lg" />
                        </Button>
                      </DropdownTrigger>
                      <DropdownMenu aria-label={t('common.actions')}>
                        <DropdownItem
                          key="delete"
                          className="text-danger"
                          color="danger"
                          startContent={<Icon icon="lucide:trash-2" />}
                          onPress={() => handleDelete(server)}
                        >
                          {t('gateway.delete')}
                        </DropdownItem>
                      </DropdownMenu>
                    </Dropdown>
                  </div>
                </div>

                {server.parsedConfig && (
                  <div className="space-y-3">
                    {(server.parsedConfig.servers || []).map((serverConfig: ServerConfig) => {
                      return (
                        <div key={serverConfig.name} className="space-y-3">
                          <div>
                            <h4 className="text-sm font-semibold truncate">{serverConfig.name}</h4>
                            <p className="text-sm text-default-500 line-clamp-2">{serverConfig.description}</p>
                          </div>

                          <div className="space-y-2">
                            <h4 className="text-sm font-semibold">{t('gateway.routing_config')}</h4>
                            <div className="flex flex-col gap-2">
                              {(server.parsedConfig?.routers ?? []).map((router: RouterConfig, idx: number) => (
                                <div key={idx} className="flex items-center gap-2">
                                  <Chip
                                    color="primary"
                                    variant="flat"
                                    size="sm"
                                    className="cursor-pointer hover:opacity-80 select-none"
                                    onClick={() => handleCopyToClipboard(router.prefix)}
                                  >
                                    {router.prefix}
                                  </Chip>
                                  <Icon icon="lucide:arrow-right" className="text-sm" />
                                  <Chip
                                    variant="flat"
                                    size="sm"
                                    className="cursor-pointer hover:opacity-80 select-none"
                                    onClick={() => handleCopyToClipboard(router.server)}
                                  >
                                    {router.server}
                                  </Chip>
                                </div>
                              ))}
                            </div>
                          </div>

                          {/* 显示MCP后端配置 */}
                          {server.parsedConfig?.mcpServers && server.parsedConfig.mcpServers.length > 0 && (
                            <div className="space-y-2">
                              <h4 className="text-sm font-semibold">{t('gateway.backend_config')}</h4>
                              <div className="flex flex-col gap-2">
                                {server.parsedConfig.mcpServers.map((mcpServer, idx) => (
                                  <div key={idx} className="flex flex-col gap-1 p-2 border border-default-200 rounded-md">
                                    <div className="flex items-center gap-2">
                                      <span className="text-sm font-medium">{mcpServer.name}</span>
                                      <Chip size="sm" variant="flat" color="warning">{mcpServer.type}</Chip>
                                    </div>
                                    {mcpServer.type === 'stdio' && (
                                      <div className="text-xs">
                                        <div className="flex items-center gap-1">
                                          <span className="font-medium">Command:</span>
                                          <code className="bg-default-100 px-1 rounded">{mcpServer.command} {mcpServer.args?.join(' ')}</code>
                                        </div>
                                        {mcpServer.env && Object.keys(mcpServer.env).length > 0 && (
                                          <div className="mt-1">
                                            <span className="font-medium">Env:</span>
                                            <div className="mt-1 pl-2">
                                              {Object.entries(mcpServer.env).map(([key, value]) => (
                                                <div key={key} className="text-xs truncate">
                                                  <span className="text-default-500">{key}:</span> {value}
                                                </div>
                                              ))}
                                            </div>
                                          </div>
                                        )}
                                      </div>
                                    )}
                                    {(mcpServer.type === 'sse' || mcpServer.type === 'streamable-http') && mcpServer.url && (
                                      <div className="text-xs">
                                        <div className="flex items-start gap-1">
                                          <span className="font-medium mt-1">URL:</span>
                                          <code className="bg-default-100 px-1 py-1 rounded break-all">{mcpServer.url}</code>
                                        </div>
                                      </div>
                                    )}
                                  </div>
                                ))}
                              </div>
                            </div>
                          )}

                          <div className="space-y-3">
                            <div>
                              <h4 className="text-sm font-semibold mb-1">{t('gateway.enabled_tools')}:</h4>
                              <div className="flex flex-wrap gap-1">
                                {serverConfig.allowedTools.map((tool: string) => (
                                  <Chip
                                    key={tool}
                                    variant="flat"
                                    color="success"
                                    size="sm"
                                    className="truncate cursor-pointer hover:opacity-80 select-none"
                                    onClick={() => handleCopyToClipboard(tool)}
                                  >
                                    {tool}
                                  </Chip>
                                ))}
                              </div>
                            </div>

                            <div>
                              <h4 className="text-sm font-semibold mb-1">{t('gateway.all_tools')}:</h4>
                              <div className="flex flex-wrap gap-1">
                                {(server.parsedConfig?.tools ?? []).map((tool: ToolConfig) => (
                                  <Chip
                                    key={tool.name}
                                    variant="flat"
                                    color="default"
                                    size="sm"
                                    className="truncate cursor-pointer hover:opacity-80 select-none"
                                    onClick={() => handleCopyToClipboard(tool.name)}
                                  >
                                    {tool.name}
                                  </Chip>
                                ))}
                              </div>
                            </div>
                          </div>
                        </div>
                      );
                    })}

                    {/* 处理只有routers和mcpServers的情况，比如proxy-mcp-exp.yaml */}
                    {(!server.parsedConfig.servers || server.parsedConfig.servers.length === 0) && (
                      <div className="space-y-3">
                        {server.parsedConfig.routers && server.parsedConfig.routers.length > 0 && (
                          <div className="space-y-2">
                            <h4 className="text-sm font-semibold">{t('gateway.routing_config')}</h4>
                            <div className="flex flex-col gap-2">
                              {server.parsedConfig.routers.map((router: RouterConfig, idx: number) => (
                                <div key={idx} className="flex items-center gap-2">
                                  <Chip
                                    color="primary"
                                    variant="flat"
                                    size="sm"
                                    className="cursor-pointer hover:opacity-80 select-none"
                                    onClick={() => handleCopyToClipboard(router.prefix)}
                                  >
                                    {router.prefix}
                                  </Chip>
                                  <Icon icon="lucide:arrow-right" className="text-sm" />
                                  <Chip
                                    variant="flat"
                                    size="sm"
                                    className="cursor-pointer hover:opacity-80 select-none"
                                    onClick={() => handleCopyToClipboard(router.server)}
                                  >
                                    {router.server}
                                  </Chip>
                                </div>
                              ))}
                            </div>
                          </div>
                        )}

                        {server.parsedConfig.mcpServers && server.parsedConfig.mcpServers.length > 0 && (
                          <div className="space-y-2">
                            <h4 className="text-sm font-semibold">{t('gateway.mcp_config')}</h4>
                            <div className="flex flex-col gap-2">
                              {server.parsedConfig.mcpServers.map((mcpServer, idx) => (
                                <div key={idx} className="flex flex-col gap-1 p-2 border border-default-200 rounded-md">
                                  <div className="flex items-center gap-2">
                                    <span className="text-sm font-medium">{mcpServer.name}</span>
                                    <Chip size="sm" variant="flat" color="warning">{mcpServer.type}</Chip>
                                  </div>
                                  {mcpServer.type === 'stdio' && (
                                    <div className="text-xs">
                                      <div className="flex items-center gap-1">
                                        <span className="font-medium">Command:</span>
                                        <code className="bg-default-100 px-1 rounded">{mcpServer.command} {mcpServer.args?.join(' ')}</code>
                                      </div>
                                      {mcpServer.env && Object.keys(mcpServer.env).length > 0 && (
                                        <div className="mt-1">
                                          <span className="font-medium">Env:</span>
                                          <div className="mt-1 pl-2">
                                            {Object.entries(mcpServer.env).map(([key, value]) => (
                                              <div key={key} className="text-xs truncate">
                                                <span className="text-default-500">{key}:</span> {value}
                                              </div>
                                            ))}
                                          </div>
                                        </div>
                                      )}
                                    </div>
                                  )}
                                  {(mcpServer.type === 'sse' || mcpServer.type === 'streamable-http') && mcpServer.url && (
                                    <div className="text-xs">
                                      <div className="flex items-start gap-1">
                                        <span className="font-medium mt-1">URL:</span>
                                        <code className="bg-default-100 px-1 py-1 rounded break-all">{mcpServer.url}</code>
                                      </div>
                                    </div>
                                  )}
                                </div>
                              ))}
                            </div>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                )}
              </CardBody>
            </Card>
          ))}
        </div>
      )}

      <Modal
        isOpen={isOpen}
        onOpenChange={onOpenChange}
        size="3xl"
        className="w-[70%] h-[70%]"
      >
        <ModalContent className="h-[70%]">
          {(onClose) => (
            <>
              <ModalHeader>{t('gateway.edit_config')}</ModalHeader>
              <ModalBody className="flex-1">
                <Editor
                  height="100%"
                  defaultLanguage="yaml"
                  value={editConfig}
                  onChange={(value) => setEditConfig(value || '')}
                  theme={isDark ? "vs-dark" : "vs"}
                  options={editorOptions}
                />
              </ModalBody>
              <ModalFooter>
                <Button color="danger" variant="light" onPress={onClose}>
                  {t('common.cancel')}
                </Button>
                <Button color="primary" onPress={handleSave}>
                  {t('common.save')}
                </Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>

      <Modal
        isOpen={isCreateOpen}
        onOpenChange={onCreateOpenChange}
        size="3xl"
        className="w-[70%] h-[70%]"
      >
        <ModalContent className="h-[70%]">
          {(onClose) => (
            <>
              <ModalHeader>{t('gateway.add_config')}</ModalHeader>
              <ModalBody className="flex-1">
                <Editor
                  height="100%"
                  defaultLanguage="yaml"
                  value={newConfig}
                  onChange={(value) => setNewConfig(value || '')}
                  theme={isDark ? "vs-dark" : "vs"}
                  options={editorOptions}
                />
              </ModalBody>
              <ModalFooter>
                <Button color="danger" variant="light" onPress={onClose}>
                  {t('common.cancel')}
                </Button>
                <Button color="primary" onPress={handleCreate}>
                  {t('gateway.add')}
                </Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>

      <Modal isOpen={isImportOpen} onOpenChange={onImportOpenChange} size="2xl">
        <ModalContent>
          <ModalHeader>{t('gateway.import_openapi')}</ModalHeader>
          <ModalBody>
            <OpenAPIImport onSuccess={handleImportSuccess} />
          </ModalBody>
          <ModalFooter>
            <Button color="danger" variant="light" onPress={() => onImportOpenChange()}>
              {t('common.cancel')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </div>
  );
}
