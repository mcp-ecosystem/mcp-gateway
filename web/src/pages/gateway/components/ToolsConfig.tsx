import { Input, Select, SelectItem, Button, Checkbox } from "@heroui/react";
import { useTranslation } from 'react-i18next';

import { GatewayConfig } from '../types';

interface ToolsConfigProps {
  parsedConfig: GatewayConfig;
  updateConfig: (newData: Partial<GatewayConfig>) => void;
}

export function ToolsConfig({
  parsedConfig,
  updateConfig
}: ToolsConfigProps) {
  const { t } = useTranslation();
  const tools = parsedConfig?.tools || [];

  const updateTool = (index: number, field: string, value: string | Array<{
    name: string;
    position: string;
    required: boolean;
    type: string;
    description: string;
    default: string;
  }>) => {
    const updatedTools = [...tools];
    const oldName = updatedTools[index].name;
    updatedTools[index] = {
      ...updatedTools[index],
      [field]: value
    };

    // If tool name changed, update server references
    if (field === 'name' && oldName !== value && parsedConfig.servers) {
      const updatedServers = parsedConfig.servers.map(server => {
        if (server.allowedTools) {
          const updatedAllowedTools = server.allowedTools.map(toolName =>
            toolName === oldName ? value as string : toolName
          );
          return { ...server, allowedTools: updatedAllowedTools };
        }
        return server;
      });
      updateConfig({ tools: updatedTools, servers: updatedServers });
    } else {
      updateConfig({ tools: updatedTools });
    }
  };

  const updateHeader = (toolIndex: number, headerIndex: number, field: 'key' | 'value', value: string) => {
    const updatedTools = [...tools];
    const tool = updatedTools[toolIndex];
    const headers = { ...tool.headers };
    const headersOrder = [...(tool.headersOrder || Object.keys(headers))];

    if (field === 'key') {
      const oldKey = headersOrder[headerIndex];
      const newKey = value;
      if (oldKey !== newKey) {
        // Update header key
        headers[newKey] = headers[oldKey];
        delete headers[oldKey];
        headersOrder[headerIndex] = newKey;
      }
    } else {
      // Update header value
      headers[headersOrder[headerIndex]] = value;
    }

    updatedTools[toolIndex] = {
      ...tool,
      headers,
      headersOrder
    };

    updateConfig({ tools: updatedTools });
  };

  const addHeader = (toolIndex: number) => {
    const updatedTools = [...tools];
    const tool = updatedTools[toolIndex];
    const headers = { ...tool.headers };
    const headersOrder = [...(tool.headersOrder || Object.keys(headers))];

    let newKey = "Content-Type";
    let count = 1;
    
    const commonHeaders = [
      "Authorization", 
      "Accept", 
      "X-API-Key", 
      "User-Agent", 
    ];
    
    for (const header of commonHeaders) {
      if (!headersOrder.includes(header)) {
        newKey = header;
        break;
      }
    }
    
    if (headersOrder.includes(newKey)) {
      while (headersOrder.includes(`X-Header-${count}`)) {
        count++;
      }
      newKey = `X-Header-${count}`;
    }
    
    headers[newKey] = "";
    headersOrder.push(newKey);

    updatedTools[toolIndex] = {
      ...tool,
      headers,
      headersOrder
    };

    updateConfig({ tools: updatedTools });
  };

  const removeHeader = (toolIndex: number, headerIndex: number) => {
    const updatedTools = [...tools];
    const tool = updatedTools[toolIndex];
    const headers = { ...tool.headers };
    const headersOrder = [...(tool.headersOrder || Object.keys(headers))];

    const keyToRemove = headersOrder[headerIndex];
    delete headers[keyToRemove];
    headersOrder.splice(headerIndex, 1);

    updatedTools[toolIndex] = {
      ...tool,
      headers,
      headersOrder
    };

    updateConfig({ tools: updatedTools });
  };

  return (
    <div className="border-t pt-4 mt-2">
      <h3 className="text-sm font-medium mb-2">{t('gateway.tools_config')}</h3>
      {tools.map((tool, index) => (
        <div key={index} className="flex flex-col gap-2 mb-4 p-3 border rounded-md">
          <Input
            label={t('gateway.tool_name')}
            value={tool.name || ""}
            onChange={(e) => updateTool(index, 'name', e.target.value)}
          />
          <Input
            label={t('gateway.description')}
            value={tool.description || ""}
            onChange={(e) => updateTool(index, 'description', e.target.value)}
          />
          <Select
            label={t('gateway.method')}
            selectedKeys={[tool.method || "GET"]}
            onChange={(e) => updateTool(index, 'method', e.target.value)}
            aria-label={t('gateway.method')}
          >
            <SelectItem key="GET">GET</SelectItem>
            <SelectItem key="POST">POST</SelectItem>
            <SelectItem key="PUT">PUT</SelectItem>
            <SelectItem key="DELETE">DELETE</SelectItem>
          </Select>
          <Input
            label={t('gateway.endpoint')}
            value={tool.endpoint || ""}
            onChange={(e) => updateTool(index, 'endpoint', e.target.value)}
          />
          
          {/* Headers 配置 */}
          <div className="mt-2 border-t pt-2">
            <h4 className="text-sm font-medium mb-2">Headers</h4>
            <div className="flex flex-col gap-2">
              {(tool.headersOrder || Object.keys(tool.headers || {})).map((key, headerIndex) => (
                <div key={headerIndex} className="flex items-center gap-2">
                  <Input
                    className="flex-1"
                    value={key}
                    onChange={(e) => updateHeader(index, headerIndex, 'key', e.target.value)}
                    placeholder="Header名称"
                  />
                  <Input
                    className="flex-1"
                    value={tool.headers?.[key] || ""}
                    onChange={(e) => updateHeader(index, headerIndex, 'value', e.target.value)}
                    placeholder="Header值"
                  />
                  <Button
                    isIconOnly
                    color="danger"
                    onPress={() => removeHeader(index, headerIndex)}
                  >
                    ✕
                  </Button>
                </div>
              ))}
              
              {/* 添加新的Header */}
              <Button
                color="primary"
                size="sm"
                className="mt-1"
                onPress={() => addHeader(index)}
              >
                添加Header
              </Button>
            </div>
          </div>

          {/* Arguments 配置 */}
          <div className="mt-2 border-t pt-2">
            <h4 className="text-sm font-medium mb-2">{t('gateway.arguments_config')}</h4>
            <div className="flex flex-col gap-2">
              {(tool.args || []).map((arg, argIndex) => (
                <div key={argIndex} className="flex flex-col gap-2 p-2 border rounded">
                  <div className="flex items-center gap-2">
                    <Input
                      className="flex-1"
                      label={t('gateway.argument_name')}
                      value={arg.name || ""}
                      onChange={(e) => {
                        const updatedArgs = [...(tool.args || [])];
                        updatedArgs[argIndex] = {
                          ...updatedArgs[argIndex],
                          name: e.target.value
                        };
                        updateTool(index, 'args', updatedArgs);
                      }}
                      placeholder={t('gateway.argument_name')}
                    />
                    <Select
                      className="flex-1"
                      label={t('gateway.argument_position')}
                      selectedKeys={[arg.position || "body"]}
                      onChange={(e) => {
                        const updatedArgs = [...(tool.args || [])];
                        updatedArgs[argIndex] = {
                          ...updatedArgs[argIndex],
                          position: e.target.value
                        };
                        updateTool(index, 'args', updatedArgs);
                      }}
                    >
                      <SelectItem key="body">{t('gateway.position_body')}</SelectItem>
                      <SelectItem key="query">{t('gateway.position_query')}</SelectItem>
                      <SelectItem key="path">{t('gateway.position_path')}</SelectItem>
                    </Select>

                    <Button
                      isIconOnly
                      color="danger"
                      onPress={() => {
                        const updatedArgs = [...(tool.args || [])];
                        updatedArgs.splice(argIndex, 1);
                        updateTool(index, 'args', updatedArgs);
                      }}
                    >
                      ✕
                    </Button>
                  </div>
                  <div className="flex items-center gap-2">
                    <Select
                      className="flex-1"
                      label={t('gateway.argument_type')}
                      selectedKeys={[arg.type || "string"]}
                      onChange={(e) => {
                        const updatedArgs = [...(tool.args || [])];
                        updatedArgs[argIndex] = {
                          ...updatedArgs[argIndex],
                          type: e.target.value
                        };
                        updateTool(index, 'args', updatedArgs);
                      }}
                    >
                      <SelectItem key="string">{t('gateway.type_string')}</SelectItem>
                      <SelectItem key="number">{t('gateway.type_number')}</SelectItem>
                      <SelectItem key="boolean">{t('gateway.type_boolean')}</SelectItem>
                      <SelectItem key="array">{t('gateway.type_array')}</SelectItem>
                      <SelectItem key="object">{t('gateway.type_object')}</SelectItem>
                    </Select>
                    <div className="flex items-center gap-2">
                      <Checkbox
                        isSelected={arg.required || false}
                        onValueChange={(isSelected) => {
                          const updatedArgs = [...(tool.args || [])];
                          updatedArgs[argIndex] = {
                            ...updatedArgs[argIndex],
                            required: isSelected
                          };
                          updateTool(index, 'args', updatedArgs);
                        }}
                      >
                        {t('gateway.argument_required')}
                      </Checkbox>
                    </div>
                  </div>
                  <Input
                    label={t('gateway.argument_description')}
                    value={arg.description || ""}
                    onChange={(e) => {
                      const updatedArgs = [...(tool.args || [])];
                      updatedArgs[argIndex] = {
                        ...updatedArgs[argIndex],
                        description: e.target.value
                      };
                      updateTool(index, 'args', updatedArgs);
                    }}
                    placeholder={t('gateway.argument_description')}
                  />
                  <Input
                    label={t('gateway.argument_default')}
                    value={arg.default || ""}
                    onChange={(e) => {
                      const updatedArgs = [...(tool.args || [])];
                      updatedArgs[argIndex] = {
                        ...updatedArgs[argIndex],
                        default: e.target.value
                      };
                      updateTool(index, 'args', updatedArgs);
                    }}
                    placeholder={t('gateway.argument_default')}
                  />
                </div>
              ))}
              
              {/* 添加新的参数 */}
              <Button
                color="primary"
                size="sm"
                className="mt-1"
                onPress={() => {
                  const updatedArgs = [...(tool.args || [])];
                  updatedArgs.push({
                    name: "",
                    position: "body",
                    required: false,
                    type: "string",
                    description: "",
                    default: ""
                  });
                  updateTool(index, 'args', updatedArgs);
                }}
              >
                {t('gateway.add_argument')}
              </Button>
            </div>
          </div>
          
          {/* Request Body */}
          <div className="mt-2 border-t pt-2">
            <h4 className="text-sm font-medium mb-2">{t('gateway.request_body')}</h4>
            <textarea
              className="w-full border rounded p-2"
              rows={5}
              value={tool.requestBody || ""}
              onChange={(e) => updateTool(index, 'requestBody', e.target.value)}
              placeholder={t('gateway.request_body_placeholder')}
            ></textarea>
          </div>
          
          {/* Response Body */}
          <div className="mt-2 border-t pt-2">
            <h4 className="text-sm font-medium mb-2">{t('gateway.response_body')}</h4>
            <textarea
              className="w-full border rounded p-2"
              rows={5}
              value={tool.responseBody || ""}
              onChange={(e) => updateTool(index, 'responseBody', e.target.value)}
              placeholder={t('gateway.response_body_placeholder')}
            ></textarea>
          </div>
        </div>
      ))}
      {/* 添加工具按钮 */}
      <Button
        color="primary"
        className="mt-2 w-full"
        onPress={() => {
          const updatedTools = [...tools];
          updatedTools.push({ 
            name: "", 
            description: "", 
            method: "GET", 
            endpoint: "",
            headers: {
              "Content-Type": "application/json"
            },
            headersOrder: ["Content-Type"],
            args: [],
            requestBody: "",
            responseBody: "{{.Response.Body}}"
          });
          updateConfig({ tools: updatedTools });
        }}
      >
        {t('gateway.add_tool')}
      </Button>
    </div>
  );
} 