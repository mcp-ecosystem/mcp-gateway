import { Input, Select, SelectItem, Button, Checkbox, Accordion, AccordionItem, Textarea } from "@heroui/react";
import { Icon } from "@iconify/react";
import { useTranslation } from 'react-i18next';

import { Gateway, ToolConfig, ServerConfig } from '../../../types/gateway';

interface ToolsConfigProps {
  parsedConfig: Gateway;
  updateConfig: (newData: Partial<Gateway>) => void;
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
      const updatedServers = parsedConfig.servers.map((server: ServerConfig) => {
        if (server.allowedTools) {
          const updatedAllowedTools = server.allowedTools.map((toolName: string) =>
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
    <div className="space-y-4">
      <Accordion variant="splitted">
        {tools.map((tool: ToolConfig, index: number) => (
          <AccordionItem 
            key={index} 
            title={tool.name || `Tool ${index + 1}`}
            subtitle={tool.description}
            startContent={
              <Icon 
                icon="lucide:wrench" 
                className="text-primary-500" 
              />
            }
          >
            <div className="p-2 space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
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
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
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
              </div>

              {/* Headers Section */}
              <div className="bg-content1 p-4 rounded-medium border border-content2">
                <div className="flex justify-between items-center mb-3">
                  <h4 className="text-md font-medium">Headers</h4>
                  <Button 
                    size="sm" 
                    color="primary" 
                    variant="flat"
                    startContent={<Icon icon="lucide:plus" />}
                    onPress={() => addHeader(index)}
                  >
                    {t('gateway.add_header')}
                  </Button>
                </div>
                
                <div className="space-y-3">
                  {(tool.headersOrder || Object.keys(tool.headers || {})).map((key: string, headerIndex: number) => (
                    <div key={headerIndex} className="flex gap-2">
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
                        variant="light"
                        className="self-end mb-1"
                        onPress={() => removeHeader(index, headerIndex)}
                      >
                        <Icon icon="lucide:x" />
                      </Button>
                    </div>
                  ))}
                </div>
              </div>

              {/* Arguments Section */}
              <div className="bg-content1 p-4 rounded-medium border border-content2">
                <div className="flex justify-between items-center mb-3">
                  <h4 className="text-md font-medium">{t('gateway.arguments_config')}</h4>
                  <Button
                    color="primary"
                    size="sm"
                    variant="flat"
                    startContent={<Icon icon="lucide:plus" />}
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

                <div className="space-y-3">
                  {(tool.args || []).map((arg: { name: string; position: string; required: boolean; type: string; description: string; default: string }, argIndex: number) => (
                    <div key={argIndex} className="flex flex-col gap-2 p-3 border border-content2 rounded-md bg-content1">
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
                          <SelectItem key="form-data">{t('gateway.type_form_data')}</SelectItem>
                        </Select>
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
                      <div className="flex justify-end">
                        <Button 
                          color="danger" 
                          variant="flat"
                          size="sm"
                          startContent={<Icon icon="lucide:trash-2" />}
                          onPress={() => {
                            const updatedArgs = [...(tool.args || [])];
                            updatedArgs.splice(argIndex, 1);
                            updateTool(index, 'args', updatedArgs);
                          }}
                        >
                          {t('gateway.remove_argument')}
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Request/Response Body */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Textarea
                  label={t('gateway.request_body')}
                  value={tool.requestBody || ""}
                  onChange={(e) => updateTool(index, 'requestBody', e.target.value)}
                  placeholder={t('gateway.request_body_placeholder')}
                  minRows={5}
                  className="font-mono text-sm"
                />
                <Textarea
                  label={t('gateway.response_body')}
                  value={tool.responseBody || ""}
                  onChange={(e) => updateTool(index, 'responseBody', e.target.value)}
                  placeholder={t('gateway.response_body_placeholder')}
                  minRows={5}
                  className="font-mono text-sm"
                />
              </div>

              <div className="flex justify-end">
                <Button 
                  color="danger" 
                  variant="flat" 
                  size="sm"
                  startContent={<Icon icon="lucide:trash-2" />}
                  onPress={() => {
                    const updatedTools = [...tools];
                    updatedTools.splice(index, 1);
                    updateConfig({ tools: updatedTools });
                  }}
                >
                  {t('gateway.remove_tool')}
                </Button>
              </div>
            </div>
          </AccordionItem>
        ))}
      </Accordion>

      {/* Add Tool Button */}
      <div className="flex justify-center">
        <Button
          color="primary"
          variant="flat"
          startContent={<Icon icon="lucide:plus" />}
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
    </div>
  );
} 