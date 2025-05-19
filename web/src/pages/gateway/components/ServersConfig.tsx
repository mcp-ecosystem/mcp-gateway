import { Input, Button, Chip } from "@heroui/react";
import { useTranslation } from 'react-i18next';

import { GatewayConfig } from '../types';

interface ServersConfigProps {
  parsedConfig: GatewayConfig;
  updateConfig: (newData: Partial<GatewayConfig>) => void;
}

export function ServersConfig({
  parsedConfig,
  updateConfig
}: ServersConfigProps) {
  const { t } = useTranslation();
  const servers = parsedConfig?.servers || [{ name: "", description: "", allowedTools: [] }];

  const updateServer = (index: number, field: 'name' | 'description', value: string) => {
    const updatedServers = [...servers];
    const oldName = updatedServers[index].name;
    updatedServers[index] = {
      ...updatedServers[index],
      [field]: value
    };

    // If server name changed, update router references
    if (field === 'name' && oldName !== value && parsedConfig.routers) {
      const updatedRouters = parsedConfig.routers.map(router => {
        if (router.server === oldName) {
          return { ...router, server: value };
        }
        return router;
      });
      updateConfig({ servers: updatedServers, routers: updatedRouters });
    } else {
      updateConfig({ servers: updatedServers });
    }
  };

  const addServer = () => {
    const newServer = {
      name: "",
      description: "",
      allowedTools: []
    };
    updateConfig({
      servers: [...servers, newServer]
    });
  };

  const removeServer = (index: number) => {
    const updatedServers = servers.filter((_, i) => i !== index);
    updateConfig({
      servers: updatedServers
    });
  };

  return (
    <div className="border-t pt-4 mt-2">
      <h3 className="text-sm font-medium mb-2">{t('gateway.server_config')}</h3>
      {servers.map((server, index) => (
        <div key={index} className="flex flex-col gap-2 mb-4 p-3 border rounded-md">
          <div className="flex justify-between items-center">
            <div className="flex-1 flex flex-row gap-4">
              <Input
                label={t('gateway.server_name')}
                value={server.name || ""}
                onChange={(e) => updateServer(index, 'name', e.target.value)}
              />
              <Input
                label={t('gateway.description')}
                value={server.description || ""}
                onChange={(e) => updateServer(index, 'description', e.target.value)}
              />
            </div>
            <Button
              color="danger"
              isIconOnly
              className="ml-2"
              onPress={() => removeServer(index)}
            >
              ✕
            </Button>
          </div>
          <div>
            <h4 className="text-sm font-medium mb-2">{t('gateway.allowed_tools')}</h4>
            <div className="flex flex-wrap gap-1">
              {server.allowedTools && server.allowedTools.map((tool, toolIndex) => (
                <Chip
                  key={toolIndex}
                  onClose={() => {
                    const updated = [...server.allowedTools];
                    updated.splice(toolIndex, 1);
                    updateConfig({
                      servers: servers.map((s, i) =>
                        i === index ? { ...s, allowedTools: updated } : s
                      )
                    });
                  }}
                >
                  {tool}
                </Chip>
              ))}
            </div>
            <div className="mt-2">
              <h4 className="text-sm font-medium mb-2">{t('gateway.add_tool')}</h4>
              <div className="flex flex-wrap gap-2">
                {(parsedConfig?.tools || [])
                  .filter(tool => !server.allowedTools?.includes(tool.name || ""))
                  .map(tool => (
                    <Button
                      key={tool.name}
                      size="sm"
                      variant="flat"
                      color="primary"
                      className="min-w-0"
                      onPress={() => {
                        if (tool.name && server.allowedTools && !server.allowedTools.includes(tool.name)) {
                          updateConfig({
                            servers: servers.map((s, i) =>
                              i === index ? {
                                ...s,
                                allowedTools: [...s.allowedTools, tool.name]
                              } : s
                            )
                          });
                        }
                      }}
                    >
                      + {tool.name || t('common.name')}
                    </Button>
                  ))
                }
                {(parsedConfig?.tools || []).length > 0 &&
                 (parsedConfig?.tools || []).every(tool => server.allowedTools?.includes(tool.name || "")) && (
                  <span className="text-sm text-gray-500">{t('gateway.tools')} {t('common.already')} {t('common.all')} {t('common.add')}</span>
                )}
                {(parsedConfig?.tools || []).length === 0 && (
                  <span className="text-sm text-gray-500">{t('gateway.tools')} {t('common.none')} {t('common.available')}</span>
                )}
              </div>
            </div>
          </div>
        </div>
      ))}
      <Button
        size="sm"
        color="primary"
        onPress={addServer}
        className="w-full"
      >
        {t('gateway.add_server')}
      </Button>
    </div>
  );
}
