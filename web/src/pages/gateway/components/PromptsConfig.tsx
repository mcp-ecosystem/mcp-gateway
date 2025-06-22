import { Input, Button, Checkbox, Accordion, AccordionItem, Textarea } from "@heroui/react";
import { Icon } from "@iconify/react";
import { useTranslation } from 'react-i18next';

import { Gateway } from '../../../types/gateway';

interface PromptArgument {
  name: string;
  description: string;
  required: boolean;
}

interface PromptsConfigProps {
  parsedConfig: Gateway;
  updateConfig: (newData: Partial<Gateway>) => void;
}

export function PromptsConfig({
  parsedConfig,
  updateConfig
}: PromptsConfigProps) {
  const { t } = useTranslation();
  const prompts = parsedConfig?.prompts || [];

  // Update a prompt at a given index
  const updatePrompt = (index: number, field: string, value: any) => {
    const updatedPrompts = [...prompts];
    updatedPrompts[index] = {
      ...updatedPrompts[index],
      [field]: value
    };
    updateConfig({ prompts: updatedPrompts });
  };

  // Update an argument for a prompt at a given index
  const updateArgument = (promptIndex: number, argIndex: number, field: keyof PromptArgument, value: string | boolean) => {
    const updatedPrompts = [...prompts];
    const updatedArgs = [...(updatedPrompts[promptIndex]?.arguments || [])];
    updatedArgs[argIndex] = { ...updatedArgs[argIndex], [field]: value };
    updatedPrompts[promptIndex] = {
      ...updatedPrompts[promptIndex],
      arguments: updatedArgs
    };
    updateConfig({ prompts: updatedPrompts });
  };

  const addArgument = (promptIndex: number) => {
    const updatedPrompts = [...prompts];
    const updatedArgs = [...(updatedPrompts[promptIndex]?.arguments || [])];
    updatedArgs.push({ name: "", description: "", required: false });
    updatedPrompts[promptIndex] = {
      ...updatedPrompts[promptIndex],
      arguments: updatedArgs
    };
    updateConfig({ prompts: updatedPrompts });
  };

  const removeArgument = (promptIndex: number, argIndex: number) => {
    const updatedPrompts = [...prompts];
    const updatedArgs = [...(updatedPrompts[promptIndex]?.arguments || [])];
    updatedArgs.splice(argIndex, 1);
    updatedPrompts[promptIndex] = {
      ...updatedPrompts[promptIndex],
      arguments: updatedArgs
    };
    updateConfig({ prompts: updatedPrompts });
  };

  // Add a new prompt
  const addPrompt = () => {
    const updatedPrompts = [
      ...prompts,
      { name: '', description: '', arguments: [] }
    ];
    updateConfig({ prompts: updatedPrompts });
  };

  // Remove a prompt
  const removePrompt = (index: number) => {
    const updatedPrompts = [...prompts];
    updatedPrompts.splice(index, 1);
    updateConfig({ prompts: updatedPrompts });
  };

  return (
    <div className="space-y-4">
      <Accordion variant="splitted">
        {prompts.map((prompt, promptIdx) => (
          <AccordionItem
            key={promptIdx}
            title={prompt.name || t('gateway.prompt') || 'Prompt'}
            subtitle={prompt.description}
            startContent={<Icon icon="lucide:message-square" className="text-primary-500" />}
          >
            <div className="p-2 space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input
                  label={t('gateway.prompt_name') || 'Prompt Name'}
                  value={prompt.name}
                  onChange={e => updatePrompt(promptIdx, 'name', e.target.value)}
                />
                <Input
                  label={t('gateway.description') || 'Description'}
                  value={prompt.description}
                  onChange={e => updatePrompt(promptIdx, 'description', e.target.value)}
                />
              </div>
              <div className="bg-content1 p-4 rounded-medium border border-content2">
                <div className="flex justify-between items-center mb-3">
                  <h4 className="text-md font-medium">{t('gateway.arguments') || 'Arguments'}</h4>
                  <Button
                    size="sm"
                    color="primary"
                    variant="flat"
                    startContent={<Icon icon="lucide:plus" />}
                    onPress={() => addArgument(promptIdx)}
                  >
                    {t('gateway.add_argument') || 'Add Argument'}
                  </Button>
                </div>
                {prompt.arguments && prompt.arguments.map((arg, argIdx) => (
                  <div key={argIdx} className="grid grid-cols-1 md:grid-cols-4 gap-2 mb-2 items-center">
                    <Input
                      label={t('gateway.argument_name') || 'Name'}
                      value={arg.name}
                      onChange={e => updateArgument(promptIdx, argIdx, 'name', e.target.value)}
                    />
                    <Input
                      label={t('gateway.argument_description') || 'Description'}
                      value={arg.description}
                      onChange={e => updateArgument(promptIdx, argIdx, 'description', e.target.value)}
                    />
                    <Checkbox
                      isSelected={arg.required}
                      onChange={e => updateArgument(promptIdx, argIdx, 'required', e.target.checked)}
                    >
                      {t('gateway.required') || 'Required'}
                    </Checkbox>
                    <Button
                      color="danger"
                      variant="flat"
                      size="sm"
                      startContent={<Icon icon="lucide:trash-2" />}
                      onPress={() => removeArgument(promptIdx, argIdx)}
                    >
                      {t('gateway.remove_argument') || 'Remove'}
                    </Button>
                  </div>
                ))}
              </div>
              {/* Prompt Response Section */}
              <div className="bg-content1 p-4 rounded-medium border border-content2 mt-4">
                <div className="flex justify-between items-center mb-3">
                  <h4 className="text-md font-medium">{t('gateway.prompt_response') || 'Prompt Response'}</h4>
                  <Button
                    size="sm"
                    color="primary"
                    variant="flat"
                    startContent={<Icon icon="lucide:plus" />}
                    onPress={() => {
                      const updatedPrompts = [...prompts];
                      const promptResponse = updatedPrompts[promptIdx].promptResponse || [];
                      promptResponse.push({ role: '', content: { type: 'text', text: '' } });
                      updatedPrompts[promptIdx] = { ...updatedPrompts[promptIdx], promptResponse };
                      updateConfig({ prompts: updatedPrompts });
                    }}
                  >
                    {t('gateway.add_prompt_response') || 'Add Response'}
                  </Button>
                </div>
                {prompt.promptResponse && prompt.promptResponse.map((ret, retIdx) => (
                  <div key={retIdx} className="grid grid-cols-1 md:grid-cols-5 gap-2 mb-2 items-center">
                    <Input
                      label={t('gateway.prompt_response_role') || 'Role'}
                      value={ret.role}
                      onChange={e => {
                        const updatedPrompts = [...prompts];
                        const promptResponse = [...(updatedPrompts[promptIdx].promptResponse || [])];
                        promptResponse[retIdx] = { ...promptResponse[retIdx], role: e.target.value };
                        updatedPrompts[promptIdx] = { ...updatedPrompts[promptIdx], promptResponse };
                        updateConfig({ prompts: updatedPrompts });
                      }}
                    />
                    <Input
                      label={t('gateway.prompt_response_type') || 'Content Type'}
                      value={ret.content.type}
                      onChange={e => {
                        const updatedPrompts = [...prompts];
                        const promptResponse = [...(updatedPrompts[promptIdx].promptResponse || [])];
                        promptResponse[retIdx] = { ...promptResponse[retIdx], content: { ...promptResponse[retIdx].content, type: e.target.value } };
                        updatedPrompts[promptIdx] = { ...updatedPrompts[promptIdx], promptResponse };
                        updateConfig({ prompts: updatedPrompts });
                      }}
                    />
                    <Textarea
                      label={t('gateway.prompt_response_text') || 'Text'}
                      value={ret.content.text}
                      minRows={3}
                      maxRows={8}
                      className="col-span-2"
                      onChange={e => {
                        const updatedPrompts = [...prompts];
                        const promptResponse = [...(updatedPrompts[promptIdx].promptResponse || [])];
                        promptResponse[retIdx] = { ...promptResponse[retIdx], content: { ...promptResponse[retIdx].content, text: e.target.value } };
                        updatedPrompts[promptIdx] = { ...updatedPrompts[promptIdx], promptResponse };
                        updateConfig({ prompts: updatedPrompts });
                      }}
                    />
                    <Button
                      color="danger"
                      variant="flat"
                      size="sm"
                      startContent={<Icon icon="lucide:trash-2" />}
                      onPress={() => {
                        const updatedPrompts = [...prompts];
                        const promptResponse = [...(updatedPrompts[promptIdx].promptResponse || [])];
                        promptResponse.splice(retIdx, 1);
                        updatedPrompts[promptIdx] = { ...updatedPrompts[promptIdx], promptResponse };
                        updateConfig({ prompts: updatedPrompts });
                      }}
                    >
                      {t('gateway.remove_prompt_response') || 'Remove'}
                    </Button>
                  </div>
                ))}
              </div>
              <div className="flex justify-end mt-2">
                <Button
                  color="danger"
                  variant="flat"
                  size="sm"
                  startContent={<Icon icon="lucide:trash-2" />}
                  onPress={() => removePrompt(promptIdx)}
                >
                  {t('gateway.remove_prompt') || 'Remove Prompt'}
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
          startContent={<Icon icon="lucide:plus" />}
          onPress={addPrompt}
        >
          {t('gateway.add_prompt') || 'Add Prompt'}
        </Button>
      </div>
    </div>
  );
}
