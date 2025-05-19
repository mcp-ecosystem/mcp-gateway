import { Card, CardBody, Button, ModalContent, ModalHeader, ModalBody, ModalFooter, useDisclosure, Chip, Dropdown, DropdownTrigger, DropdownMenu, DropdownItem, Autocomplete, AutocompleteItem, Tabs, Tab, Table, TableHeader, TableColumn, TableBody, TableRow, TableCell, Modal } from "@heroui/react";
import { Icon } from '@iconify/react';
import yaml from 'js-yaml';
import { configureMonacoYaml } from 'monaco-yaml';
import React from 'react';
import { useTranslation } from 'react-i18next';

import { getMCPServers, createMCPServer, updateMCPServer, deleteMCPServer, exportMCPServer, syncMCPServers, getUserAuthorizedTenants, getTenant } from '../../services/api';
import type { Gateway } from '../../types/gateway';
import { toast } from '../../utils/toast';

import { ConfigEditor } from './components/ConfigEditor';
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
  const [viewMode, setViewMode] = React.useState<string>('card');
  const [isDark, setIsDark] = React.useState(() => {
    return document.documentElement.classList.contains('dark');
  });
  const [isRoutingModalOpen, setIsRoutingModalOpen] = React.useState(false);
  const [isToolsModalOpen, setIsToolsModalOpen] = React.useState(false);
  const [currentModalServer, setCurrentModalServer] = React.useState<Gateway | null>(null);

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
        // Router prefix must start with tenant prefix followed by a slash and not be empty
        if (!routerPrefix.startsWith(tenantPrefix + '/')) {
          toast.error(t('errors.router_prefix_error'), {
            duration: 3000,
          });
          return false;
        }

        // Check if there is content after tenant prefix
        const remainingPath = routerPrefix.slice(tenantPrefix.length + 1);
        if (!remainingPath) {
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

   const handleExport = async (server: Gateway) => {
    try {
      toast.success(t('gateway.exporting'));
      await exportMCPServer(server);
      toast.success(t('gateway.export_success'));
    } catch {
      toast.error(t('gateway.export_failed'));
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
    <div className="container mx-auto p-4 pb-10 h-[calc(100vh-5rem)] flex flex-col overflow-y-scroll scrollbar-hide">
      <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold">{t('gateway.title')}</h1>
        <div className="flex gap-2">
          <Button
            color="primary"
            onPress={onCreateOpen}
            startContent={<Icon icon="material-symbols:add" />}
            aria-label={t('gateway.add')}
          >
            {t('gateway.add')}
          </Button>
          <Button
            color="secondary"
            variant="flat"
            onPress={onImportOpen}
            startContent={<Icon icon="material-symbols:upload" />}
            className="bg-purple-500 hover:bg-purple-600 text-white"
            aria-label={t('gateway.import_openapi')}
          >
            {t('gateway.import_openapi')}
          </Button>
          <Button
            color="default"
            onPress={handleSync}
            isLoading={isLoading}
            startContent={<Icon icon="material-symbols:sync" />}
            aria-label={t('gateway.sync')}
          >
            {t('gateway.sync')}
          </Button>
        </div>
      </div>

      <div className="flex justify-between items-center mb-4">
        <div className="max-w-md">
        <label className="block text-sm font-medium mb-1">{t('gateway.select_tenant')}</label>

        {/* Display selected tenants */}
        <div className="flex flex-wrap gap-1 mb-2">
          {selectedTenants.map(tenant => (
            <Chip
              key={tenant.id}
              onClose={() => handleRemoveTenant(tenant.id)}
              variant="flat"
              aria-label={`${tenant.name} (${tenant.prefix})`}
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
          aria-label={t('gateway.search_tenant')}
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

        <Tabs
          aria-label={t('gateway.view_mode')}
          selectedKey={viewMode}
          onSelectionChange={(key) => setViewMode(key as string)}
          size="sm"
          classNames={{
            tabList: "bg-default-100 p-1 rounded-lg"
          }}
        >
          <Tab
            key="card"
            title={
              <div className="flex items-center gap-1">
                <Icon icon="material-symbols:grid-view" className="text-lg" />
                <span>{t('gateway.card_view')}</span>
              </div>
            }
          />
          <Tab
            key="table"
            title={
              <div className="flex items-center gap-1">
                <Icon icon="material-symbols:table-rows" className="text-lg" />
                <span>{t('gateway.table_view')}</span>
              </div>
            }
          />
        </Tabs>
      </div>

      <div className="flex-1">
      {isLoading ? (
        <div className="flex justify-center items-center h-32">
          <Icon icon="lucide:loader-2" className="animate-spin text-2xl" />
        </div>
        ) : viewMode === 'card' ? (
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
                        <DropdownItem
                          key="export"
                          className="text-green-500"
                          color="primary"
                          startContent={<Icon icon="lucide:download" />}
                          onPress={() => handleExport(server)}
                        >
                          {t('gateway.export')}
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
                                    aria-label={`${t('common.copy')} ${router.prefix}`}
                                  >
                                    {router.prefix}
                                  </Chip>
                                  <Icon icon="lucide:arrow-right" className="text-sm" />
                                  <Chip
                                    variant="flat"
                                    size="sm"
                                    className="cursor-pointer hover:opacity-80 select-none"
                                    onClick={() => handleCopyToClipboard(router.server)}
                                    aria-label={`${t('common.copy')} ${router.server}`}
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
                                      <Chip size="sm" variant="flat" color="warning" aria-label={`Type: ${mcpServer.type}`}>
                                        {mcpServer.type}
                                      </Chip>
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
                                    aria-label={`${t('common.copy')} ${tool}`}
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
                                    aria-label={`${t('common.copy')} ${tool.name}`}
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
                                    aria-label={`${t('common.copy')} ${router.prefix}`}
                                  >
                                    {router.prefix}
                                  </Chip>
                                  <Icon icon="lucide:arrow-right" className="text-sm" />
                                  <Chip
                                    variant="flat"
                                    size="sm"
                                    className="cursor-pointer hover:opacity-80 select-none"
                                    onClick={() => handleCopyToClipboard(router.server)}
                                    aria-label={`${t('common.copy')} ${router.server}`}
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
                                    <Chip size="sm" variant="flat" color="warning" aria-label={`Type: ${mcpServer.type}`}>
                                      {mcpServer.type}
                                    </Chip>
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
        ) : (
          <Table aria-label={t('gateway.table_view')}>
            <TableHeader>
              <TableColumn width="25%">{t('gateway.name')}</TableColumn>
              <TableColumn width="15%">{t('gateway.description')}</TableColumn>
              <TableColumn width="25%">{t('gateway.routing')}</TableColumn>
              <TableColumn width="25%">{t('gateway.tools')}</TableColumn>
              <TableColumn width="10%">{t('common.actions')}</TableColumn>
            </TableHeader>
            <TableBody>
              {(parsedMCPServers || []).map((server) => (
                <TableRow key={server.name}>
                  <TableCell className="font-medium truncate max-w-[15%]">{server.name}</TableCell>
                  <TableCell className="max-w-[25%]">
                    {server.parsedConfig && server.parsedConfig.servers && server.parsedConfig.servers.length > 0 ? (
                      <div className="line-clamp-2 overflow-hidden overflow-ellipsis">
                        {server.parsedConfig.servers[0].description}
                      </div>
                    ) : (
                      <span className="text-gray-400">{t('gateway.no_description')}</span>
                    )}
                  </TableCell>
                  <TableCell className="max-w-[25%] overflow-hidden">
                    <Button
                      variant="light"
                      size="sm"
                      className="w-full text-left justify-start"
                      endContent={<Icon icon="lucide:external-link" className="text-sm" />}
                      onPress={() => {
                        setCurrentModalServer(server);
                        setIsRoutingModalOpen(true);
                      }}
                    >
                      {`${server.parsedConfig?.routers?.length || 0} ${t('gateway.routes')}`}
                    </Button>
                  </TableCell>
                  <TableCell className="max-w-[25%] overflow-hidden">
                    <Button
                      variant="light"
                      size="sm"
                      className="w-full text-left justify-start"
                      endContent={<Icon icon="lucide:external-link" className="text-sm" />}
                      onPress={() => {
                        setCurrentModalServer(server);
                        setIsToolsModalOpen(true);
                      }}
                    >
                      {server.parsedConfig?.servers && server.parsedConfig.servers.length > 0
                        ? `${server.parsedConfig.servers[0].allowedTools.length} ${t('gateway.enabled')} / ${server.parsedConfig.tools?.length || 0} ${t('gateway.total')}`
                        : `${server.parsedConfig?.tools?.length || 0} ${t('gateway.total')}`}
                    </Button>
                  </TableCell>
                  <TableCell>
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
                      <Button
                        isIconOnly
                        color="danger"
                        variant="light"
                        size="sm"
                        onPress={() => handleDelete(server)}
                        aria-label={t('gateway.delete')}
                      >
                        <Icon icon="lucide:trash-2" className="text-lg" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>

      <Modal
        isOpen={isOpen}
        onOpenChange={onOpenChange}
        size="3xl"
        className="w-[70%] h-[80%]"
        scrollBehavior="inside"
      >
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>{t('gateway.edit_config')}</ModalHeader>
              <ModalBody>
                <ConfigEditor 
                  config={editConfig}
                  onChange={(value) => setEditConfig(value)}
                  isDark={isDark} 
                  editorOptions={editorOptions}
                  isEditing={true}
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
        className="w-[70%] h-[80%]"
        scrollBehavior="inside"
      >
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>{t('gateway.add_config')}</ModalHeader>
              <ModalBody className="overflow-y-auto">
                <ConfigEditor 
                  config={newConfig}
                  onChange={(value) => setNewConfig(value)}
                  isDark={isDark} 
                  editorOptions={editorOptions}
                  isEditing={false}
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

      <Modal
        isOpen={isRoutingModalOpen}
        onClose={() => setIsRoutingModalOpen(false)}
        size="2xl"
        scrollBehavior="inside"
      >
        <ModalContent>
          {() => (
            <>
              <ModalHeader className="flex flex-col gap-1">
                {currentModalServer?.name} - {t('gateway.routing_config')}
              </ModalHeader>
              <ModalBody>
                <div className="space-y-3">
                  <div className="space-y-2">
                    <h4 className="text-sm font-semibold">{t('gateway.routing_config')}</h4>
                    <div className="space-y-2 w-full">
                      {(currentModalServer?.parsedConfig?.routers || []).map((router: RouterConfig, idx: number) => (
                        <div key={idx} className="flex items-center gap-2 flex-wrap">
                          <Chip
                            color="primary"
                            variant="flat"
                            size="sm"
                            className="cursor-pointer hover:opacity-80 select-none"
                            onClick={() => handleCopyToClipboard(router.prefix)}
                            aria-label={`${t('common.copy')} ${router.prefix}`}
                          >
                            {router.prefix}
                          </Chip>
                          <Icon icon="lucide:arrow-right" className="text-sm" />
                          <Chip
                            variant="flat"
                            size="sm"
                            className="cursor-pointer hover:opacity-80 select-none"
                            onClick={() => handleCopyToClipboard(router.server)}
                            aria-label={`${t('common.copy')} ${router.server}`}
                          >
                            {router.server}
                          </Chip>
                        </div>
                      ))}
                    </div>
                  </div>

                  {currentModalServer?.parsedConfig?.mcpServers && currentModalServer.parsedConfig.mcpServers.length > 0 && (
                    <div className="space-y-2 mt-4 pt-4 border-t">
                      <h4 className="text-sm font-semibold">{t('gateway.backend_config')}</h4>
                      <div className="space-y-2">
                        {currentModalServer.parsedConfig.mcpServers.map((mcpServer, idx) => (
                          <div key={idx} className="flex flex-col gap-1 p-2 border border-default-200 rounded-md">
                            <div className="flex items-center gap-2 flex-wrap">
                              <span className="text-sm font-medium">{mcpServer.name}</span>
                              <Chip size="sm" variant="flat" color="warning" aria-label={`Type: ${mcpServer.type}`}>
                                {mcpServer.type}
                              </Chip>
                            </div>
                            {mcpServer.type === 'stdio' && (
                              <div className="text-xs">
                                <div className="flex items-center gap-1 flex-wrap">
                                  <span className="font-medium">Command:</span>
                                  <code className="bg-default-100 px-1 rounded break-all">{mcpServer.command} {mcpServer.args?.join(' ')}</code>
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
              </ModalBody>
              <ModalFooter>
                <Button color="primary" size="sm" onPress={() => setIsRoutingModalOpen(false)}>
                  {t('common.close')}
                </Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>

      <Modal
        isOpen={isToolsModalOpen}
        onClose={() => setIsToolsModalOpen(false)}
        size="2xl"
        scrollBehavior="inside"
      >
        <ModalContent>
          {() => (
            <>
              <ModalHeader className="flex flex-col gap-1">
                {currentModalServer?.name} - {t('gateway.tools')}
              </ModalHeader>
              <ModalBody>
                {currentModalServer?.parsedConfig?.servers && currentModalServer.parsedConfig.servers.length > 0 && (
                  <div className="space-y-6">
                    <div>
                      <h4 className="text-sm font-semibold mb-2">{t('gateway.enabled_tools')}</h4>
                      <div className="flex flex-wrap gap-1">
                        {currentModalServer.parsedConfig.servers[0].allowedTools.map((tool: string) => (
                          <Chip
                            key={tool}
                            variant="flat"
                            color="success"
                            size="sm"
                            className="truncate cursor-pointer hover:opacity-80 select-none"
                            onClick={() => handleCopyToClipboard(tool)}
                            aria-label={`${t('common.copy')} ${tool}`}
                          >
                            {tool}
                          </Chip>
                        ))}
                      </div>
                    </div>

                    <div className="mt-4 pt-4 border-t">
                      <h4 className="text-sm font-semibold mb-2">{t('gateway.all_tools')}</h4>
                      <div className="flex flex-wrap gap-1">
                        {(currentModalServer.parsedConfig?.tools ?? []).map((tool: ToolConfig) => (
                          <Chip
                            key={tool.name}
                            variant="flat"
                            color="default"
                            size="sm"
                            className="truncate cursor-pointer hover:opacity-80 select-none"
                            onClick={() => handleCopyToClipboard(tool.name)}
                            aria-label={`${t('common.copy')} ${tool.name}`}
                          >
                            {tool.name}
                          </Chip>
                        ))}
                      </div>
                    </div>
                  </div>
                )}
              </ModalBody>
              <ModalFooter>
                <Button color="primary" size="sm" onPress={() => setIsToolsModalOpen(false)}>
                  {t('common.close')}
                </Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </div>
  );
}
