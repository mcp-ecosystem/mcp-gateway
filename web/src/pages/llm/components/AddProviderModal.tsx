import {
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Button,
  Input,
  Textarea,
  Card,
  CardBody,
  Chip,
  Divider,
  Switch,
  Slider
} from '@heroui/react';
import { Plus, X } from 'lucide-react';
import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';

import ProviderIcon from '@/components/ProviderIcon';
import { BUILTIN_PROVIDERS, getProviderTemplate, getProviderDefaultConfig } from '@/config/llm-providers-adapter';
import { CreateLLMProviderForm } from '@/types/llm';
import { toast } from '@/utils/toast';

interface AddProviderModalProps {
  isOpen: boolean;
  onClose: () => void;
  onAdd: (data: CreateLLMProviderForm) => void;
}

const AddProviderModal: React.FC<AddProviderModalProps> = ({
  isOpen,
  onClose,
  onAdd
}) => {
  const { t } = useTranslation();
  const [selectedProvider, setSelectedProvider] = useState<string | null>(null);
  const [formData, setFormData] = useState<CreateLLMProviderForm>({
    name: '',
    description: '',
    config: {}
  });

  const handleSelectProvider = (providerId: string) => {
    setSelectedProvider(providerId);
    const template = getProviderTemplate(providerId);
    
    if (template) {
      setFormData({
        name: template.name,
        description: template.description,
        config: getProviderDefaultConfig(providerId),
        models: template.defaultModels
      });
    }
  };

  const handleCustomProvider = () => {
    setSelectedProvider('custom');
    setFormData({
      name: '',
      description: '',
      config: {
        temperature: 0.7,
        topP: 1.0,
        maxTokens: 2048,
        timeout: 30000,
        fetchOnClient: false
      }
    });
  };

  const handleSubmit = () => {
    if (!formData.name.trim()) {
      toast.error(t('llm.nameRequired'));
      return;
    }

    onAdd(formData);
    toast.success(t('llm.addSuccess', { name: formData.name }));
    handleClose();
  };

  const handleClose = () => {
    setSelectedProvider(null);
    setFormData({
      name: '',
      description: '',
      config: {}
    });
    onClose();
  };

  const isCustom = selectedProvider === 'custom';

  return (
    <Modal
      isOpen={isOpen}
      onClose={handleClose}
      size="2xl"
      scrollBehavior="inside"
      className="max-h-[90vh]"
    >
      <ModalContent>
        <ModalHeader className="flex items-center gap-2">
          <Plus className="w-5 h-5" />
          {t('llm.addProvider')}
        </ModalHeader>
        
        <ModalBody>
          {!selectedProvider ? (
            // 选择提供商
            <div className="space-y-4">
              <div>
                <h3 className="font-semibold mb-3">{t('llm.selectProvider')}</h3>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  {BUILTIN_PROVIDERS.map((provider) => (
                    <Card
                      key={provider.id}
                      isPressable
                      onPress={() => handleSelectProvider(provider.id)}
                      className="hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
                    >
                      <CardBody className="p-4">
                        <div className="flex items-center gap-3">
                          <ProviderIcon
                            providerId={provider.id}
                            name={provider.name}
                            size={28}
                            fallbackUrl={provider.logo}
                          />
                          <div className="flex-1">
                            <h4 className="font-medium">{provider.name}</h4>
                            <p className="text-sm text-gray-600 dark:text-gray-400 line-clamp-2">
                              {provider.description}
                            </p>
                          </div>
                        </div>
                      </CardBody>
                    </Card>
                  ))}
                </div>
              </div>

              <Divider />

              <div>
                <h3 className="font-semibold mb-3">{t('llm.customProvider')}</h3>
                <Card
                  isPressable
                  onPress={handleCustomProvider}
                  className="hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
                >
                  <CardBody className="p-4">
                    <div className="flex items-center gap-3">
                      <div className="w-8 h-8 rounded-full bg-gradient-to-r from-blue-500 to-purple-600 flex items-center justify-center">
                        <Plus className="w-4 h-4 text-white" />
                      </div>
                      <div>
                        <h4 className="font-medium">{t('llm.addCustomProvider')}</h4>
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                          {t('llm.customProviderDesc')}
                        </p>
                      </div>
                    </div>
                  </CardBody>
                </Card>
              </div>
            </div>
          ) : (
            // 配置提供商
            <div className="space-y-4">
              <div className="flex items-center gap-3">
                <Button
                  size="sm"
                  variant="flat"
                  onPress={() => setSelectedProvider(null)}
                >
                  ← {t('common.back')}
                </Button>
                <div className="flex items-center gap-2">
                  {!isCustom && (
                    <ProviderIcon
                      providerId={selectedProvider}
                      name={formData.name}
                      size={24}
                      fallbackUrl={BUILTIN_PROVIDERS.find(p => p.id === selectedProvider)?.logo}
                    />
                  )}
                  <h3 className="font-semibold">
                    {isCustom ? t('llm.customProvider') : formData.name}
                  </h3>
                  {isCustom && (
                    <Chip size="sm" color="secondary" variant="flat">
                      {t('llm.custom')}
                    </Chip>
                  )}
                </div>
              </div>

              <div className="space-y-4">
                <Input
                  label={t('llm.providerName')}
                  placeholder={t('llm.providerNamePlaceholder')}
                  value={formData.name}
                  onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                  isRequired
                  isReadOnly={!isCustom}
                />

                <Textarea
                  label={t('llm.description')}
                  placeholder={t('llm.descriptionPlaceholder')}
                  value={formData.description || ''}
                  onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
                  maxRows={3}
                />

                {isCustom && (
                  <>
                    <Input
                      label={t('llm.baseURL')}
                      placeholder="https://api.example.com/v1"
                      value={(formData.config.baseURL as string) || ''}
                      onChange={(e) => setFormData(prev => ({ 
                        ...prev, 
                        config: { ...prev.config, baseURL: e.target.value }
                      }))}
                    />

                    <Input
                      label={t('llm.apiKey')}
                      placeholder={t('llm.apiKeyPlaceholder')}
                      type="password"
                      value={(formData.config.apiKey as string) || ''}
                      onChange={(e) => setFormData(prev => ({ 
                        ...prev, 
                        config: { ...prev.config, apiKey: e.target.value }
                      }))}
                    />

                    <div className="flex items-center justify-between">
                      <div>
                        <p className="font-medium">{t('llm.fetchOnClient')}</p>
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                          {t('llm.fetchOnClientDesc')}
                        </p>
                      </div>
                      <Switch
                        isSelected={formData.config.fetchOnClient as boolean}
                        onValueChange={(checked) => setFormData(prev => ({ 
                          ...prev, 
                          config: { ...prev.config, fetchOnClient: checked }
                        }))}
                      />
                    </div>

                    <Divider />

                    <div className="space-y-4">
                      <h3 className="font-semibold">{t('llm.modelParameters')}</h3>
                      
                      <div>
                        <div className="flex items-center justify-between mb-2">
                          <label className="font-medium">{t('llm.temperature')}</label>
                          <span className="text-sm text-gray-600">{(formData.config.temperature as number) || 0.7}</span>
                        </div>
                        <Slider
                          step={0.1}
                          minValue={0}
                          maxValue={2}
                          value={(formData.config.temperature as number) || 0.7}
                          onChange={(value) => setFormData(prev => ({ 
                            ...prev, 
                            config: { ...prev.config, temperature: value as number }
                          }))}
                          className="w-full"
                        />
                      </div>

                      <div>
                        <div className="flex items-center justify-between mb-2">
                          <label className="font-medium">{t('llm.topP')}</label>
                          <span className="text-sm text-gray-600">{(formData.config.topP as number) || 1.0}</span>
                        </div>
                        <Slider
                          step={0.1}
                          minValue={0}
                          maxValue={1}
                          value={(formData.config.topP as number) || 1.0}
                          onChange={(value) => setFormData(prev => ({ 
                            ...prev, 
                            config: { ...prev.config, topP: value as number }
                          }))}
                          className="w-full"
                        />
                      </div>

                      <Input
                        label={t('llm.maxTokens')}
                        placeholder="2048"
                        type="number"
                        value={String((formData.config.maxTokens as number) || 2048)}
                        onChange={(e) => setFormData(prev => ({ 
                          ...prev, 
                          config: { ...prev.config, maxTokens: parseInt(e.target.value) || 2048 }
                        }))}
                      />

                      <Input
                        label={t('llm.timeout')}
                        placeholder="30000"
                        type="number"
                        value={String((formData.config.timeout as number) || 30000)}
                        onChange={(e) => setFormData(prev => ({ 
                          ...prev, 
                          config: { ...prev.config, timeout: parseInt(e.target.value) || 30000 }
                        }))}
                      />
                    </div>
                  </>
                )}

                {!isCustom && (
                  <div className="bg-blue-50 dark:bg-blue-900/20 p-4 rounded-lg">
                    <p className="text-sm text-blue-800 dark:text-blue-200">
                      {t('llm.providerConfigNote')}
                    </p>
                  </div>
                )}
              </div>
            </div>
          )}
        </ModalBody>
        
        <ModalFooter>
          <Button
            variant="flat"
            onPress={handleClose}
            startContent={<X className="w-4 h-4" />}
          >
            {t('common.cancel')}
          </Button>
          
          {selectedProvider && (
            <Button
              color="primary"
              onPress={handleSubmit}
              startContent={<Plus className="w-4 h-4" />}
              isDisabled={!formData.name.trim()}
            >
              {t('llm.add')}
            </Button>
          )}
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
};

export default AddProviderModal;