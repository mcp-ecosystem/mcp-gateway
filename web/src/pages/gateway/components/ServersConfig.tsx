import { Input, Button, Chip, Accordion, AccordionItem } from "@heroui/react";
import { useTranslation } from 'react-i18next';

import LocalIcon from '@/components/LocalIcon';
import { Gateway, RouterConfig, ServerConfig } from '@/types/gateway';

interface ServersConfigProps {
  parsedConfig: Gateway;
  updateConfig: (newData: Partial<Gateway>) => void;
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
      const updatedRouters = parsedConfig.routers.map((router: RouterConfig) => {
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
    const newServer: ServerConfig = {
      name: "",
      description: "",
      allowedTools: []
    };
    updateConfig({
      servers: [...servers, newServer]
    });
  };

  const removeServer = (index: number) => {
    const updatedServers = servers.filter((_: ServerConfig, i: number) => i !== index);
    updateConfig({
      servers: updatedServers
    });
  };

  return (
    <div className="space-y-4">
      <Accordion variant="splitted">
        {servers.map((server: ServerConfig, index: number) => (
          <AccordionItem 
            key={index} 
            title={server.name || `Server ${index + 1}`}
            subtitle={server.description}
            startContent={
              <LocalIcon icon="lucide:server" className="text-primary-500" />
            }
          >
            <div className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
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

              <div className="bg-content1 p-4 rounded-medium border border-content2">
                <div className="flex justify-between items-center mb-3">
                  <h4 className="text-md font-medium">{t('gateway.allowed_tools')}</h4>
                </div>
                <div className="flex flex-wrap gap-1">
                  {(server.allowedTools || []).map((tool: string, toolIndex: number) => (
                    <Chip
                      key={toolIndex}
                      onClose={() => {
                        const updated = [...(server.allowedTools || [])];
                        updated.splice(toolIndex, 1);
                        updateConfig({
                          servers: servers.map((s: ServerConfig, i: number) =>
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
                      .filter((tool: { name?: string }) => tool.name && !(server.allowedTools || []).includes(tool.name))
                      .map((tool: { name?: string }) => (
                        <Button
                          key={tool.name}
                          size="sm"
                          variant="flat"
                          color="primary"
                          className="min-w-0"
                          onPress={() => {
                            if (tool.name && !(server.allowedTools || []).includes(tool.name)) {
                              const updatedServer: ServerConfig = {
                                ...server,
                                allowedTools: [...(server.allowedTools || []), tool.name]
                              };
                              updateConfig({
                                servers: servers.map((s: ServerConfig, i: number) =>
                                  i === index ? updatedServer : s
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
                     (parsedConfig?.tools || []).every((tool: { name?: string }) => tool.name && (server.allowedTools || []).includes(tool.name)) && (
                      <span className="text-sm text-default-500">{t('gateway.tools_already_all_added')}</span>
                    )}
                    {(parsedConfig?.tools || []).length === 0 && (
                      <span className="text-sm text-default-500">{t('gateway.tools_none_available')}</span>
                    )}
                  </div>
                </div>
              </div>

              <div className="flex justify-end">
                <Button 
                  color="danger" 
                  variant="flat" 
                  size="sm"
                  startContent={<LocalIcon icon="lucide:trash-2" />}
                  onPress={() => removeServer(index)}
                >
                  {t('gateway.remove_server')}
                </Button>
              </div>
            </div>
          </AccordionItem>
        ))}
      </Accordion>

      <div className="flex justify-center">
        <Button
          color="primary"
          variant="flat"
          startContent={<LocalIcon icon="lucide:plus" />}
          onPress={addServer}
        >
          {t('gateway.add_server')}
        </Button>
      </div>
    </div>
  );
}
