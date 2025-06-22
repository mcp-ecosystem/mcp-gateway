
export interface PromptArgument {
  name: string;
  description: string;
  required: boolean;
}

export interface PromptResponseContent {
  type: string;
  text: string;
}

export interface PromptResponse {
  role: string;
  content: PromptResponseContent;
}

export interface PromptConfig {
  name: string;
  description: string;
  arguments?: PromptArgument[];
  promptResponse?: PromptResponse[];
}
