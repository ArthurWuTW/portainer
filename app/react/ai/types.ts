export interface LLMConfig {
  baseUrl: string;
  apiKey: string;
  model: string;
  temperature: number;
  maxTokens: number;
}

export interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
}

export type ToolEventStatus = 'running' | 'done' | 'error';

export interface ToolEvent {
  name: string;
  arguments?: string;
  status: ToolEventStatus;
}

export const DEFAULT_LLM_CONFIG: LLMConfig = {
  baseUrl: 'http://localhost:8000/v1',
  apiKey: '',
  model: 'Qwen3-27B',
  temperature: 0.2,
  maxTokens: 2048,
};
