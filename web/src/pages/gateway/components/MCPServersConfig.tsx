import { Input, Select, SelectItem, Button } from "@heroui/react";
import { useState, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';

import { GatewayConfig } from '../types';

interface MCPServersConfigProps {
  parsedConfig: GatewayConfig;
  updateConfig: (newData: Partial<GatewayConfig>) => void;
}

export function MCPServersConfig({
  parsedConfig,
  updateConfig
}: MCPServersConfigProps) {
  const { t } = useTranslation();
  const mcpServers = useMemo(() => 
    parsedConfig?.mcpServers || [{ type: "stdio", name: "", command: "", args: [], env: {} }],
    [parsedConfig?.mcpServers]
  );
  const [commandInputs, setCommandInputs] = useState<{ [key: number]: string }>({});

  // Initialize command inputs when mcpServers changes
  useEffect(() => {
    const initialInputs = mcpServers.reduce((acc, server, index) => {
      acc[index] = `${server.command || ''} ${server.args?.join(' ') || ''}`.trim();
      return acc;
    }, {} as { [key: number]: string });
    setCommandInputs(initialInputs);
  }, [mcpServers]);

  const updateServer = (index: number, field: string, value: string) => {
    const updatedServers = [...mcpServers];
    const oldName = updatedServers[index].name;
    
    if (field === 'command') {
      // Split the command string by whitespace and update both command and args
      const parts = value.trim().split(/\s+/);
      updatedServers[index] = {
        ...updatedServers[index],
        command: parts[0] || '',
        args: parts.slice(1)
      };
    } else {
      updatedServers[index] = {
        ...updatedServers[index],
        [field]: value
      };
    }

    // If server name changed, update router references
    if (field === 'name' && oldName !== value && parsedConfig.routers) {
      const updatedRouters = parsedConfig.routers.map(router => {
        if (router.server === oldName) {
          return { ...router, server: value };
        }
        return router;
      });
      updateConfig({ mcpServers: updatedServers, routers: updatedRouters });
    } else {
      updateConfig({ mcpServers: updatedServers });
    }
  };

  const handleCommandInputChange = (index: number, value: string) => {
    setCommandInputs(prev => ({
      ...prev,
      [index]: value
    }));
  };

  const handleCommandInputBlur = (index: number) => {
    updateServer(index, 'command', commandInputs[index]);
  };

  const updateEnvVariable = (serverIndex: number, envIndex: number, field: 'key' | 'value', value: string) => {
    const updatedServers = [...mcpServers];
    const server = updatedServers[serverIndex];
    const env = { ...server.env };
    const envKeys = Object.keys(env);
    const key = envKeys[envIndex];

    if (field === 'key') {
      if (key !== value) {
        env[value] = env[key];
        delete env[key];
      }
    } else {
      env[key] = value;
    }

    updatedServers[serverIndex] = {
      ...server,
      env
    };

    updateConfig({ mcpServers: updatedServers });
  };

  const addEnvVariable = (serverIndex: number) => {
    const updatedServers = [...mcpServers];
    const server = updatedServers[serverIndex];
    const env = { ...server.env };

    let newKey = "API_KEY";
    let count = 1;

    const commonEnvVars = [
      "AUTH_TOKEN",
      "OPENAI_API_KEY",
      "ANTHROPIC_API_KEY",
      "GITHUB_TOKEN",
      "MODEL_NAME"
    ];

    const existingKeys = Object.keys(env);

    for (const envVar of commonEnvVars) {
      if (!existingKeys.includes(envVar)) {
        newKey = envVar;
        break;
      }
    }

    if (existingKeys.includes(newKey)) {
      while (existingKeys.includes(`ENV_VAR_${count}`)) {
        count++;
      }
      newKey = `ENV_VAR_${count}`;
    }

    env[newKey] = "";

    updatedServers[serverIndex] = {
      ...server,
      env
    };

    updateConfig({ mcpServers: updatedServers });
  };

  const removeEnvVariable = (serverIndex: number, envIndex: number) => {
    const updatedServers = [...mcpServers];
    const server = updatedServers[serverIndex];
    const env = { ...server.env };
    const key = Object.keys(env)[envIndex];
    delete env[key];

    updatedServers[serverIndex] = {
      ...server,
      env
    };

    updateConfig({ mcpServers: updatedServers });
  };

  const addServer = () => {
    const newServer = {
      type: "stdio",
      name: "",
      command: "",
      args: [],
      env: {}
    };
    updateConfig({
      mcpServers: [...mcpServers, newServer]
    });
  };

  const removeServer = (index: number) => {
    const updatedServers = mcpServers.filter((_, i) => i !== index);
    updateConfig({
      mcpServers: updatedServers
    });
  };

  return (
    <div className="border-t pt-4 mt-2">
      <h3 className="text-sm font-medium mb-2">{t('gateway.mcp_server_config')}</h3>
      {mcpServers.map((server, index) => (
        <div key={index} className="flex flex-col gap-2 mb-4 p-3 border rounded-md">
          <div className="flex justify-between items-center">
            <div className="flex-1">
              <Input
                label={t('gateway.server_name')}
                value={server.name || ""}
                onChange={(e) => updateServer(index, 'name', e.target.value)}
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

          <Select
            label={t('gateway.mcp_type')}
            selectedKeys={[server.type || "stdio"]}
            onChange={(e) => updateServer(index, 'type', e.target.value)}
            aria-label={t('gateway.mcp_type')}
          >
            <SelectItem key="stdio">stdio</SelectItem>
            <SelectItem key="sse">sse</SelectItem>
            <SelectItem key="streamable-http">streamable-http</SelectItem>
          </Select>

          <Select
            label={t('gateway.startup_policy')}
            selectedKeys={[server.policy || "onDemand"]}
            onChange={(e) => updateServer(index, 'policy', e.target.value)}
            aria-label={t('gateway.startup_policy')}
          >
            <SelectItem key="onDemand">{t('gateway.policy_on_demand')}</SelectItem>
            <SelectItem key="onStart">{t('gateway.policy_on_start')}</SelectItem>
          </Select>

          {(server.type === 'stdio' || !server.type) && (
            <>
              <Input
                label={t('gateway.command')}
                value={commandInputs[index] || ''}
                onChange={(e) => handleCommandInputChange(index, e.target.value)}
                onBlur={() => handleCommandInputBlur(index)}
                placeholder="command arg1 arg2 arg3"
                type="text"
                inputMode="text"
              />

              <div className="mt-2">
                <h4 className="text-sm font-medium mb-2">{t('gateway.env_variables')}</h4>
                <div className="flex flex-col gap-2">
                  {Object.entries(server.env || {}).map(([key, value], envIndex) => (
                    <div key={envIndex} className="flex items-center gap-2">
                      <Input
                        className="flex-1"
                        value={key}
                        onChange={(e) => updateEnvVariable(index, envIndex, 'key', e.target.value)}
                        placeholder="环境变量名称"
                      />
                      <Input
                        className="flex-1"
                        value={String(value)}
                        onChange={(e) => updateEnvVariable(index, envIndex, 'value', e.target.value)}
                        placeholder="环境变量值"
                      />
                      <Button
                        color="danger"
                        isIconOnly
                        onPress={() => removeEnvVariable(index, envIndex)}
                      >
                        ✕
                      </Button>
                    </div>
                  ))}

                  <Button
                    color="primary"
                    size="sm"
                    className="mt-1"
                    onPress={() => addEnvVariable(index)}
                  >
                    添加环境变量
                  </Button>
                </div>
              </div>
            </>
          )}

          {(server.type === 'sse' || server.type === 'streamable-http') && (
            <Input
              label={t('gateway.url')}
              value={server.url || ''}
              onChange={(e) => updateServer(index, 'url', e.target.value)}
            />
          )}
        </div>
      ))}
      <Button
        size="sm"
        color="primary"
        onPress={addServer}
        className="w-full"
      >
        {t('gateway.add_mcp_server')}
      </Button>
    </div>
  );
}
