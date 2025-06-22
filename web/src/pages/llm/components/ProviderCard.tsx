import {
  Card,
  CardBody,
  CardHeader,
  Button,
  Switch,
  Chip,
  Tooltip,
  Divider,
  useDisclosure,
  Avatar
} from '@heroui/react';
import { 
  Settings, 
  Trash2, 
  CheckCircle, 
  XCircle, 
  Globe,
  Book,
  ChevronDown,
  ChevronUp
} from 'lucide-react';
import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';

import ProviderConfigModal from './ProviderConfigModal';

import { LLMProvider, UpdateLLMProviderForm } from '@/types/llm';
import { toast } from '@/utils/toast';


interface ProviderCardProps {
  provider: LLMProvider;
  onUpdate: (data: UpdateLLMProviderForm) => void;
  onDelete: () => void;
  onToggle: (enabled: boolean) => void;
  onTest: () => Promise<{ success: boolean; error?: string }>;
}

const ProviderCard: React.FC<ProviderCardProps> = ({
  provider,
  onUpdate,
  onDelete,
  onToggle,
  onTest
}) => {
  const { t } = useTranslation();
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<{ success: boolean; error?: string } | null>(null);
  const [expanded, setExpanded] = useState(false);
  const { isOpen: isConfigOpen, onOpen: onConfigOpen, onClose: onConfigClose } = useDisclosure();

  const handleTest = async () => {
    setTesting(true);
    setTestResult(null);
    
    try {
      const result = await onTest();
      setTestResult(result);
      
      if (result.success) {
        toast.success(t('llm.testSuccess'));
      } else {
        toast.error(t('llm.testFailed', { error: result.error }));
      }
    } catch (error) {
      const result = { 
        success: false, 
        error: error instanceof Error ? error.message : 'Unknown error'
      };
      setTestResult(result);
      toast.error(t('llm.testError', { error: result.error }));
    } finally {
      setTesting(false);
    }
  };

  const handleDelete = () => {
    if (window.confirm(t('llm.deleteConfirm', { name: provider.name }))) {
      onDelete();
      toast.success(t('llm.deleteSuccess', { name: provider.name }));
    }
  };

  const isCustomProvider = provider.id.startsWith('custom_');
  const enabledModels = provider.models.filter(model => model.enabled);

  return (
    <>
      <Card className={`transition-all duration-200 ${provider.enabled ? 'border-success' : ''}`}>
        <CardHeader>
          <div className="flex items-center justify-between w-full">
            <div className="flex items-center gap-3">
              <Avatar
                src={provider.logo}
                name={provider.name}
                size="sm"
                fallback={provider.name.charAt(0).toUpperCase()}
              />
              <div>
                <div className="flex items-center gap-2">
                  <h3 className="font-semibold text-lg">{provider.name}</h3>
                  {isCustomProvider && (
                    <Chip size="sm" color="secondary" variant="flat">
                      {t('llm.custom')}
                    </Chip>
                  )}
                </div>
                {provider.description && (
                  <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                    {provider.description}
                  </p>
                )}
              </div>
            </div>

            <div className="flex items-center gap-2">
              {/* 连接状态 */}
              {testResult && (
                <Tooltip content={testResult.success ? t('llm.connected') : testResult.error}>
                  {testResult.success ? (
                    <CheckCircle className="w-5 h-5 text-success" />
                  ) : (
                    <XCircle className="w-5 h-5 text-danger" />
                  )}
                </Tooltip>
              )}

              {/* 启用开关 */}
              <Switch
                isSelected={provider.enabled}
                onValueChange={onToggle}
                color="success"
                size="sm"
              />
            </div>
          </div>
        </CardHeader>

        <CardBody>
          <div className="space-y-4">
            {/* 基本信息 */}
            <div className="flex flex-wrap items-center gap-2">
              <Chip size="sm" variant="flat">
                {t('llm.modelsCount', { count: enabledModels.length })}
              </Chip>
              
              {Boolean(provider.config.baseURL) && (
                <Chip size="sm" variant="flat" color="primary">
                  {t('llm.customEndpoint')}
                </Chip>
              )}
              
              {Boolean(provider.config.fetchOnClient) && (
                <Chip size="sm" variant="flat" color="warning">
                  {t('llm.clientMode')}
                </Chip>
              )}
            </div>

            {/* 展开/折叠的详细信息 */}
            {expanded && (
              <div className="space-y-3">
                <Divider />
                
                {/* 配置信息 */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
                  {Boolean(provider.config.baseURL) && (
                    <div>
                      <span className="font-medium">{t('llm.baseURL')}:</span>
                      <p className="text-gray-600 dark:text-gray-400 truncate">
                        {provider.config.baseURL as string}
                      </p>
                    </div>
                  )}
                  
                  {Boolean(provider.config.apiKey) && (
                    <div>
                      <span className="font-medium">{t('llm.apiKey')}:</span>
                      <p className="text-gray-600 dark:text-gray-400">
                        {'*'.repeat(Math.min((provider.config.apiKey as string).length, 20))}
                      </p>
                    </div>
                  )}
                  
                  <div>
                    <span className="font-medium">{t('llm.temperature')}:</span>
                    <p className="text-gray-600 dark:text-gray-400">
                      {(provider.config.temperature as number) ?? 0.7}
                    </p>
                  </div>
                  
                  <div>
                    <span className="font-medium">{t('llm.maxTokens')}:</span>
                    <p className="text-gray-600 dark:text-gray-400">
                      {(provider.config.maxTokens as number) ?? 2048}
                    </p>
                  </div>
                </div>

                {/* 模型列表 */}
                {enabledModels.length > 0 && (
                  <div>
                    <p className="font-medium mb-2">{t('llm.enabledModels')}:</p>
                    <div className="flex flex-wrap gap-1">
                      {enabledModels.map((model) => (
                        <Chip key={model.id} size="sm" variant="bordered">
                          {model.name}
                        </Chip>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}

            <Divider />

            {/* 操作按钮 */}
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Button
                  size="sm"
                  variant="flat"
                  onPress={() => setExpanded(!expanded)}
                  startContent={expanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
                >
                  {expanded ? t('common.collapse') : t('common.expand')}
                </Button>

                {/* 外部链接 */}
                {!isCustomProvider && (
                  <div className="flex items-center gap-1">
                    <Tooltip content={t('llm.website')}>
                      <Button
                        size="sm"
                        variant="light"
                        isIconOnly
                        as="a"
                        href={`https://${provider.id}.com`}
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        <Globe className="w-4 h-4" />
                      </Button>
                    </Tooltip>
                    
                    <Tooltip content={t('llm.documentation')}>
                      <Button
                        size="sm"
                        variant="light"
                        isIconOnly
                        as="a"
                        href="#"
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        <Book className="w-4 h-4" />
                      </Button>
                    </Tooltip>
                  </div>
                )}
              </div>

              <div className="flex items-center gap-2">
                <Button
                  size="sm"
                  color="primary"
                  variant="flat"
                  onPress={handleTest}
                  isLoading={testing}
                  startContent={!testing && <CheckCircle className="w-4 h-4" />}
                  isDisabled={!provider.enabled}
                >
                  {testing ? t('llm.testing') : t('llm.test')}
                </Button>

                <Button
                  size="sm"
                  color="default"
                  variant="flat"
                  onPress={onConfigOpen}
                  startContent={<Settings className="w-4 h-4" />}
                >
                  {t('llm.configure')}
                </Button>

                {isCustomProvider && (
                  <Button
                    size="sm"
                    color="danger"
                    variant="flat"
                    onPress={handleDelete}
                    startContent={<Trash2 className="w-4 h-4" />}
                  >
                    {t('common.delete')}
                  </Button>
                )}
              </div>
            </div>
          </div>
        </CardBody>
      </Card>

      <ProviderConfigModal
        isOpen={isConfigOpen}
        onClose={onConfigClose}
        provider={provider}
        onUpdate={onUpdate}
      />
    </>
  );
};

export default ProviderCard;