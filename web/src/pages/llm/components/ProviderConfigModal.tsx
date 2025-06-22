import {
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Button,
  Input,
  Switch,
  Slider,
  Tabs,
  Tab,
  Card,
  CardBody,
  Chip,
  Divider
} from '@heroui/react';
import { Eye, EyeOff, Settings, Save, X } from 'lucide-react';
import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';

import { LLMProvider, UpdateLLMProviderForm } from '@/types/llm';
import { toast } from '@/utils/toast';

interface ProviderConfigModalProps {
  isOpen: boolean;
  onClose: () => void;
  provider: LLMProvider;
  onUpdate: (data: UpdateLLMProviderForm) => void;
}

const ProviderConfigModal: React.FC<ProviderConfigModalProps> = ({
  isOpen,
  onClose,
  provider,
  onUpdate
}) => {
  const { t } = useTranslation();
  const [config, setConfig] = useState(provider.config);
  const [models, setModels] = useState(provider.models);
  const [showApiKey, setShowApiKey] = useState(false);
  const [activeTab, setActiveTab] = useState('config');

  useEffect(() => {
    if (isOpen) {
      setConfig(provider.config);
      setModels(provider.models);
    }
  }, [isOpen, provider]);

  const handleSave = () => {
    const updateData: UpdateLLMProviderForm = {
      config,
      models
    };

    onUpdate(updateData);
    toast.success(t('llm.updateSuccess'));
    onClose();
  };

  const handleCancel = () => {
    setConfig(provider.config);
    setModels(provider.models);
    onClose();
  };

  const toggleModelEnabled = (modelId: string, enabled: boolean) => {
    setModels(prev => 
      prev.map(model => 
        model.id === modelId ? { ...model, enabled } : model
      )
    );
  };

  const { settings } = provider;

  return (
    <Modal
      isOpen={isOpen}
      onClose={handleCancel}
      size="3xl"
      scrollBehavior="inside"
      className="max-h-[90vh]"
    >
      <ModalContent>
        <ModalHeader className="flex items-center gap-2">
          <Settings className="w-5 h-5" />
          {t('llm.configureProvider', { name: provider.name })}
        </ModalHeader>
        
        <ModalBody>
          <Tabs
            selectedKey={activeTab}
            onSelectionChange={(key) => setActiveTab(key as string)}
            className="w-full"
          >
            <Tab key="config" title={t('llm.configuration')}>
              <div className="space-y-4">
                {/* API 密钥 */}
                {settings.showApiKey && (
                  <div>
                    <Input
                      label={t('llm.apiKey')}
                      placeholder={t('llm.apiKeyPlaceholder')}
                      value={String(config.apiKey || '')}
                      onChange={(e) => setConfig((prev: Record<string, unknown>) => ({ ...prev, apiKey: e.target.value }))}
                      type={showApiKey ? 'text' : 'password'}
                      isRequired={settings.apiKeyRequired}
                      endContent={
                        <Button
                          size="sm"
                          variant="light"
                          isIconOnly
                          onPress={() => setShowApiKey(!showApiKey)}
                        >
                          {showApiKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                        </Button>
                      }
                    />
                  </div>
                )}

                {/* Base URL */}
                {settings.showBaseURL && (
                  <div>
                    <Input
                      label={t('llm.baseURL')}
                      placeholder={t('llm.baseURLPlaceholder')}
                      value={String(config.baseURL || '')}
                      onChange={(e) => setConfig((prev: Record<string, unknown>) => ({ ...prev, baseURL: e.target.value }))}
                      isRequired={settings.baseURLRequired}
                    />
                  </div>
                )}

                {/* Organization */}
                {settings.showOrganization && (
                  <div>
                    <Input
                      label={t('llm.organization')}
                      placeholder={t('llm.organizationPlaceholder')}
                      value={String(config.organization || '')}
                      onChange={(e) => setConfig((prev: Record<string, unknown>) => ({ ...prev, organization: e.target.value }))}
                    />
                  </div>
                )}

                <Divider />

                {/* 客户端获取 */}
                {settings.allowClientFetch && (
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="font-medium">{t('llm.fetchOnClient')}</p>
                      <p className="text-sm text-gray-600 dark:text-gray-400">
                        {t('llm.fetchOnClientDesc')}
                      </p>
                    </div>
                    <Switch
                      isSelected={Boolean(config.fetchOnClient)}
                      onValueChange={(checked) => setConfig((prev: Record<string, unknown>) => ({ ...prev, fetchOnClient: checked }))}
                    />
                  </div>
                )}

                <Divider />

                {/* 模型参数 */}
                <div className="space-y-4">
                  <h3 className="font-semibold">{t('llm.modelParameters')}</h3>
                  
                  {/* Temperature */}
                  {settings.showTemperature && (
                    <div>
                      <div className="flex items-center justify-between mb-2">
                        <label className="font-medium">{t('llm.temperature')}</label>
                        <span className="text-sm text-gray-600">{Number(config.temperature) || 0.7}</span>
                      </div>
                      <Slider
                        step={0.1}
                        minValue={0}
                        maxValue={2}
                        value={Number(config.temperature) || 0.7}
                        onChange={(value) => setConfig((prev: Record<string, unknown>) => ({ ...prev, temperature: value as number }))}
                        className="w-full"
                      />
                    </div>
                  )}

                  {/* Top P */}
                  {settings.showTopP && (
                    <div>
                      <div className="flex items-center justify-between mb-2">
                        <label className="font-medium">{t('llm.topP')}</label>
                        <span className="text-sm text-gray-600">{Number(config.topP) || 1.0}</span>
                      </div>
                      <Slider
                        step={0.1}
                        minValue={0}
                        maxValue={1}
                        value={Number(config.topP) || 1.0}
                        onChange={(value) => setConfig((prev: Record<string, unknown>) => ({ ...prev, topP: value as number }))}
                        className="w-full"
                      />
                    </div>
                  )}

                  {/* Max Tokens */}
                  {settings.showMaxTokens && (
                    <div>
                      <Input
                        type="number"
                        label={t('llm.maxTokens')}
                        placeholder="2048"
                        value={config.maxTokens?.toString() || ''}
                        onChange={(e) => setConfig((prev: Record<string, unknown>) => ({ 
                          ...prev, 
                          maxTokens: parseInt(e.target.value) || undefined 
                        }))}
                      />
                    </div>
                  )}

                  {/* Timeout */}
                  {settings.showTimeout && (
                    <div>
                      <Input
                        type="number"
                        label={t('llm.timeout')}
                        placeholder="30000"
                        value={config.timeout?.toString() || ''}
                        onChange={(e) => setConfig((prev: Record<string, unknown>) => ({ 
                          ...prev, 
                          timeout: parseInt(e.target.value) || undefined 
                        }))}
                        endContent={<span className="text-sm text-gray-500">ms</span>}
                      />
                    </div>
                  )}
                </div>
              </div>
            </Tab>

            <Tab key="models" title={t('llm.models')}>
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="font-semibold">{t('llm.availableModels')}</h3>
                  <Chip size="sm" color="primary">
                    {models.filter(m => m.enabled).length} / {models.length} {t('llm.enabled')}
                  </Chip>
                </div>

                {models.length === 0 ? (
                  <Card>
                    <CardBody className="text-center py-8">
                      <p className="text-gray-600 dark:text-gray-400">
                        {t('llm.noModels')}
                      </p>
                    </CardBody>
                  </Card>
                ) : (
                  <div className="space-y-2 max-h-64 overflow-y-auto">
                    {models.map((model) => (
                      <Card key={model.id} className="transition-all duration-200">
                        <CardBody className="py-3">
                          <div className="flex items-center justify-between">
                            <div className="flex-1">
                              <div className="flex items-center gap-2 mb-1">
                                <h4 className="font-medium">{model.name}</h4>
                                {model.capabilities.vision && (
                                  <Chip size="sm" color="secondary" variant="flat">Vision</Chip>
                                )}
                                {model.capabilities.toolCalls && (
                                  <Chip size="sm" color="success" variant="flat">Tools</Chip>
                                )}
                                {model.capabilities.reasoning && (
                                  <Chip size="sm" color="warning" variant="flat">Reasoning</Chip>
                                )}
                              </div>
                              
                              {model.description && (
                                <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                                  {model.description}
                                </p>
                              )}
                              
                              <div className="flex items-center gap-4 text-xs text-gray-500">
                                {model.contextWindow && (
                                  <span>{t('llm.contextWindow')}: {model.contextWindow.toLocaleString()}</span>
                                )}
                                {model.maxTokens && (
                                  <span>{t('llm.maxOutput')}: {model.maxTokens.toLocaleString()}</span>
                                )}
                                {model.pricing && (
                                  <span>
                                    ${model.pricing.input}/{model.pricing.output} per {model.pricing.unit.replace('per_', '').replace('_', ' ')}
                                  </span>
                                )}
                              </div>
                            </div>
                            
                            <Switch
                              isSelected={model.enabled}
                              onValueChange={(enabled) => toggleModelEnabled(model.id, enabled)}
                              size="sm"
                            />
                          </div>
                        </CardBody>
                      </Card>
                    ))}
                  </div>
                )}
              </div>
            </Tab>
          </Tabs>
        </ModalBody>
        
        <ModalFooter>
          <Button
            variant="flat"
            onPress={handleCancel}
            startContent={<X className="w-4 h-4" />}
          >
            {t('common.cancel')}
          </Button>
          <Button
            color="primary"
            onPress={handleSave}
            startContent={<Save className="w-4 h-4" />}
          >
            {t('common.save')}
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
};

export default ProviderConfigModal;