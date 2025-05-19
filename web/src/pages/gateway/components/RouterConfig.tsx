import { Input, Select, SelectItem, Button, Switch, Chip } from "@heroui/react";
import { useState } from "react";
import { useTranslation } from 'react-i18next';

import { GatewayConfig, CorsConfig, Tenant } from '../types';

interface RouterConfigProps {
  parsedConfig: GatewayConfig;
  updateConfig: (newData: Partial<GatewayConfig>) => void;
  tenants: Tenant[];
}

export function RouterConfig({
  parsedConfig,
  updateConfig,
  tenants,
}: RouterConfigProps) {
  const { t } = useTranslation();
  const selectedTenant = tenants.find(t => t.name === parsedConfig?.tenant);
  const routers = parsedConfig?.routers || [{ server: "", prefix: "/" }];

  // Add state for input values
  const [originInput, setOriginInput] = useState("");
  const [headerInput, setHeaderInput] = useState("");
  const [exposeHeaderInput, setExposeHeaderInput] = useState("");

  const updateRouter = (index: number, field: string, value: string) => {
    const updatedRouters = [...routers];
    updatedRouters[index] = {
      ...updatedRouters[index],
      [field]: value
    };
    updateConfig({ routers: updatedRouters });
  };

  const renderCorsConfig = (router: { cors?: Record<string, unknown> }, index: number) => {
    const corsConfig = router.cors as CorsConfig;
    if (!corsConfig) return null;

    const updateCors = (updates: Partial<CorsConfig>) => {
      const updatedCors = { ...corsConfig, ...updates };
      const updatedRouters = routers.map((r, i) =>
        i === index ? { ...r, cors: updatedCors } : r
      );
      updateConfig({ routers: updatedRouters });
    };

    const addCorsItem = (field: keyof CorsConfig, value: string) => {
      if (!value?.trim()) return;
      const currentValues = corsConfig[field] as string[] || [];
      if (!currentValues.includes(value.trim())) {
        updateCors({
          [field]: [...currentValues, value.trim()]
        });
      }
    };

    const removeCorsItem = (field: keyof CorsConfig, itemIndex: number) => {
      const currentValues = corsConfig[field] as string[] || [];
      updateCors({
        [field]: currentValues.filter((_, i) => i !== itemIndex)
      });
    };

    return (
      <div className="mt-2 pl-4 border-l-2 border-gray-200">
        {/* 允许的源 */}
        <div className="mb-3">
          <h4 className="text-sm font-medium mb-1">{t('gateway.allow_origins')}</h4>
          <div className="flex flex-wrap gap-1 mb-1">
            {(corsConfig.allowOrigins || []).map((origin: string, originIndex: number) => (
              <Chip
                key={originIndex}
                onClose={() => removeCorsItem('allowOrigins', originIndex)}
              >
                {origin}
              </Chip>
            ))}
          </div>
          <div className="flex gap-2">
            <Input
              size="sm"
              placeholder="例如: https://example.com 或 *"
              className="flex-1"
              value={originInput}
              onChange={(e) => setOriginInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  addCorsItem('allowOrigins', originInput);
                  setOriginInput('');
                }
              }}
            />
            <Button
              size="sm"
              onPress={() => {
                addCorsItem('allowOrigins', originInput);
                setOriginInput('');
              }}
            >
              {t('common.add')}
            </Button>
          </div>
        </div>

        {/* 允许的方法 */}
        <div className="mb-3">
          <h4 className="text-sm font-medium mb-1">{t('gateway.allow_methods')}</h4>
          <div className="flex flex-wrap gap-1 mb-1">
            {(corsConfig.allowMethods || []).map((method: string, methodIndex: number) => (
              <Chip
                key={methodIndex}
                onClose={() => removeCorsItem('allowMethods', methodIndex)}
              >
                {method}
              </Chip>
            ))}
          </div>
          <div className="flex gap-2">
            <Select
              size="sm"
              className="flex-1"
              id={`method-select-${index}`}
              aria-label={t('gateway.http_method')}
              onChange={(e) => addCorsItem('allowMethods', e.target.value)}
            >
              {['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS', 'HEAD', 'PATCH'].map(method => (
                <SelectItem key={method}>{method}</SelectItem>
              ))}
            </Select>
          </div>
        </div>

        {/* 允许的头部 */}
        <div className="mb-3">
          <h4 className="text-sm font-medium mb-1">{t('gateway.allow_headers')}</h4>
          <div className="flex flex-wrap gap-1 mb-1">
            {(corsConfig.allowHeaders || []).map((header: string, headerIndex: number) => (
              <Chip
                key={headerIndex}
                onClose={() => removeCorsItem('allowHeaders', headerIndex)}
              >
                {header}
              </Chip>
            ))}
          </div>
          <div className="flex gap-2">
            <Input
              size="sm"
              placeholder="例如: Content-Type"
              className="flex-1"
              value={headerInput}
              onChange={(e) => setHeaderInput(e.target.value)}
              list={`common-headers-${index}`}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  addCorsItem('allowHeaders', headerInput);
                  setHeaderInput('');
                }
              }}
            />
            <datalist id={`common-headers-${index}`}>
              <option value="Content-Type" />
              <option value="Authorization" />
              <option value="X-Requested-With" />
              <option value="Accept" />
              <option value="Origin" />
              <option value="Mcp-Session-Id" />
            </datalist>
            <Button
              size="sm"
              onPress={() => {
                addCorsItem('allowHeaders', headerInput);
                setHeaderInput('');
              }}
            >
              {t('common.add')}
            </Button>
          </div>
        </div>

        {/* 暴露的头部 */}
        <div className="mb-3">
          <h4 className="text-sm font-medium mb-1">{t('gateway.expose_headers')}</h4>
          <div className="flex flex-wrap gap-1 mb-1">
            {(corsConfig.exposeHeaders || []).map((header: string, headerIndex: number) => (
              <Chip
                key={headerIndex}
                onClose={() => removeCorsItem('exposeHeaders', headerIndex)}
              >
                {header}
              </Chip>
            ))}
          </div>
          <div className="flex gap-2">
            <Input
              size="sm"
              placeholder="例如: Content-Length"
              className="flex-1"
              value={exposeHeaderInput}
              onChange={(e) => setExposeHeaderInput(e.target.value)}
              list={`common-expose-headers-${index}`}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  addCorsItem('exposeHeaders', exposeHeaderInput);
                  setExposeHeaderInput('');
                }
              }}
            />
            <datalist id={`common-expose-headers-${index}`}>
              <option value="Content-Length" />
              <option value="Mcp-Session-Id" />
              <option value="X-Rate-Limit" />
            </datalist>
            <Button
              size="sm"
              onPress={() => {
                addCorsItem('exposeHeaders', exposeHeaderInput);
                setExposeHeaderInput('');
              }}
            >
              {t('common.add')}
            </Button>
          </div>
        </div>

        {/* 允许携带凭证 */}
        <div className="mb-3 flex items-center gap-2">
          <Switch
            size="sm"
            isSelected={Boolean(corsConfig.allowCredentials)}
            onValueChange={(isSelected) => updateCors({ allowCredentials: isSelected })}
          />
          <span className="text-sm">{t('gateway.credentials')}</span>
        </div>
      </div>
    );
  };

  return (
    <div className="border-t pt-4 mt-2">
      <h3 className="text-sm font-medium mb-2">{t('gateway.router_config')}</h3>
      {routers.map((router, index) => (
        <div key={index} className="flex flex-col gap-2 mb-4 p-3 border rounded-md">
          <div className="flex gap-2">
            <Input
              label={t('gateway.prefix')}
              value={(router.prefix || "").replace(selectedTenant?.prefix || "", "")}
              startContent={
                <div className="pointer-events-none flex items-center">
                  <span className="text-default-400 text-small">{selectedTenant?.prefix}</span>
                </div>
              }
              onChange={(e) => {
                const pathPart = e.target.value.trim();
                const fullPrefix = `${selectedTenant?.prefix}${pathPart}`;
                updateRouter(index, 'prefix', fullPrefix);
              }}
              className="flex-1"
            />
            <Select
              label={t('gateway.server')}
              selectedKeys={router.server ? [router.server] : []}
              className="flex-1"
              aria-label={t('gateway.server')}
              onChange={(e) => updateRouter(index, 'server', e.target.value)}
            >
              <>
                {(parsedConfig?.servers || []).map(server => (
                  <SelectItem key={server.name}>
                    {server.name}
                  </SelectItem>
                ))}
                {(parsedConfig?.mcpServers || []).map(server => (
                  <SelectItem key={server.name}>
                    {server.name}
                  </SelectItem>
                ))}
              </>
            </Select>
            <Button
              isIconOnly
              color="danger"
              className="self-end mb-2"
              onPress={() => {
                if (routers.length > 1) {
                  const updatedRouters = [...routers];
                  updatedRouters.splice(index, 1);
                  updateConfig({ routers: updatedRouters });
                }
              }}
              isDisabled={routers.length <= 1}
            >
              ✕
            </Button>
          </div>

          {/* CORS配置部分 */}
          <div className="mt-3">
            <div className="flex items-center gap-2">
              <Switch
                size="sm"
                isSelected={Boolean(router.cors)}
                onValueChange={(isSelected) => {
                  if (isSelected) {
                    updateConfig({
                      routers: routers.map((r, i) =>
                        i === index ? {
                          ...r,
                          cors: {
                            allowOrigins: ['*'],
                            allowMethods: ['GET', 'POST', 'PUT', 'OPTIONS'],
                            allowHeaders: ['Content-Type', 'Authorization', 'Mcp-Session-Id'],
                            exposeHeaders: ['Mcp-Session-Id'],
                            allowCredentials: true
                          }
                        } : r
                      )
                    });
                  } else {
                    const updatedRouters = [...routers];
                    const { cors: _, ...restRouter } = updatedRouters[index];
                    updatedRouters[index] = restRouter;
                    updateConfig({ routers: updatedRouters });
                  }
                }}
              />
              <span className="text-sm font-medium">{t('gateway.enable_cors')}</span>
            </div>

            {router.cors && renderCorsConfig(router, index)}
          </div>
        </div>
      ))}
      {/* 添加路由按钮 */}
      <Button
        color="primary"
        className="mt-2 w-full"
        onPress={() => {
          const updatedRouters = [...routers];
          const serverName = parsedConfig?.servers?.[0]?.name || parsedConfig?.mcpServers?.[0]?.name || "";

          updatedRouters.push({
            server: serverName,
            prefix: selectedTenant?.prefix + '/' + Math.random().toString(36).substring(2, 6)
          });
          updateConfig({ routers: updatedRouters });
        }}
      >
        {t('common.add')}
      </Button>
    </div>
  );
}
