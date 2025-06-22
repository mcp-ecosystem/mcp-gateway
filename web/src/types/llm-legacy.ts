import { AIChatModelCard } from './ai-model';

// Legacy types to support old LLM system - keeping them separate from new lobe-chat types

export interface LLMModel {
  id: string;
  name: string;
  description?: string;
  contextWindow?: number;
  maxTokens?: number;
  pricing?: {
    input: number;
    output: number;
    unit: 'per_1k_tokens' | 'per_1m_tokens';
  };
  capabilities: {
    vision: boolean;
    toolCalls: boolean;
    streaming: boolean;
    reasoning: boolean;
  };
  enabled?: boolean;
  isCustom?: boolean;
  ownedBy?: string;
}

export interface FetchedModel {
  id: string;
  object?: string;
  created?: number;
  owned_by?: string;
}

export interface ProviderSettings {
  showApiKey: boolean;
  showBaseURL: boolean;
  showOrganization: boolean;
  showTemperature: boolean;
  showTopP: boolean;
  showMaxTokens: boolean;
  showTimeout: boolean;
  allowClientFetch: boolean;
  defaultClientFetch: boolean;
  apiKeyRequired: boolean;
  baseURLRequired: boolean;
  checkModel?: string;
}

export interface BuiltinProviderTemplate {
  id: string;
  name: string;
  logo?: string;
  description: string;
  website?: string;
  documentation?: string;
  settings: ProviderSettings;
  defaultModels: LLMModel[];
}

export interface LLMProvider {
  id: string;
  name: string;
  logo?: string;
  description: string;
  enabled: boolean;
  config: Record<string, unknown>;
  models: LLMModel[];
  settings: ProviderSettings;
}

export interface CreateLLMProviderForm {
  name: string;
  description?: string;
  config: Record<string, unknown>;
  models?: LLMModel[];
}

export interface UpdateLLMProviderForm {
  name?: string;
  description?: string;
  enabled?: boolean;
  config?: Record<string, unknown>;
  models?: LLMModel[];
}

// Convert lobe-chat AIChatModelCard to our LLMModel format
export const convertToLLMModel = (aiModel: AIChatModelCard, providerId: string): LLMModel => {
  return {
    id: aiModel.id,
    name: aiModel.displayName || aiModel.id,
    description: aiModel.description,
    contextWindow: aiModel.contextWindowTokens,
    maxTokens: aiModel.maxOutput,
    pricing: aiModel.pricing ? {
      input: aiModel.pricing.input || 0,
      output: aiModel.pricing.output || 0,
      unit: (aiModel.pricing.input && aiModel.pricing.input < 1) ? 'per_1k_tokens' : 'per_1m_tokens'
    } : {
      input: 0,
      output: 0,
      unit: 'per_1k_tokens'
    },
    capabilities: {
      vision: aiModel.abilities?.vision || false,
      toolCalls: aiModel.abilities?.functionCall || true,
      streaming: true, // assume all models support streaming
      reasoning: aiModel.abilities?.reasoning || false
    },
    enabled: aiModel.enabled || false,
    isCustom: false,
    ownedBy: providerId
  };
};