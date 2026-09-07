import { useState } from 'react';

import { Button } from '@@/buttons';

import { useAIChatStore } from './aiChatStore';
import type { LLMConfig } from './types';

const fieldClass =
  'w-full rounded border border-gray-3 bg-widget-color px-3 py-1.5 text-sm text-gray-10 outline-none focus:border-blue-1 th-dark:border-gray-11 th-dark:bg-gray-12 th-dark:text-gray-1 th-dark:focus:border-blue-1';

const labelClass =
  'mb-1 block text-xs font-medium text-gray-8 th-dark:text-gray-warm-7';

export function LLMSettingsForm() {
  const { llmConfig, setLLMConfig } = useAIChatStore();
  const [draft, setDraft] = useState<LLMConfig>(llmConfig);

  function update(patch: Partial<LLMConfig>) {
    setDraft((d) => ({ ...d, ...patch }));
  }

  function handleSave() {
    setLLMConfig(draft);
  }

  return (
    <div className="space-y-3" data-cy="ai-chat-settings-form">
      <div>
        <label className={labelClass} htmlFor="ai-llm-baseurl">
          Base URL
        </label>
        <input
          id="ai-llm-baseurl"
          data-cy="ai-chat-baseurl-input"
          className={fieldClass}
          value={draft.baseUrl}
          onChange={(e) => update({ baseUrl: e.target.value })}
          placeholder="http://localhost:8000/v1"
        />
      </div>

      <div>
        <label className={labelClass} htmlFor="ai-llm-apikey">
          API Key (optional)
        </label>
        <input
          id="ai-llm-apikey"
          data-cy="ai-chat-apikey-input"
          type="password"
          className={fieldClass}
          value={draft.apiKey}
          onChange={(e) => update({ apiKey: e.target.value })}
          placeholder="Leave blank if no key is required"
        />
      </div>

      <div>
        <label className={labelClass} htmlFor="ai-llm-model">
          Model
        </label>
        <input
          id="ai-llm-model"
          data-cy="ai-chat-model-input"
          className={fieldClass}
          value={draft.model}
          onChange={(e) => update({ model: e.target.value })}
          placeholder="Qwen3-27B"
        />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className={labelClass} htmlFor="ai-llm-temperature">
            Temperature
          </label>
          <input
            id="ai-llm-temperature"
            data-cy="ai-chat-temperature-input"
            type="number"
            min={0}
            max={2}
            step={0.1}
            className={fieldClass}
            value={draft.temperature}
            onChange={(e) =>
              update({ temperature: Number(e.target.value) || 0 })
            }
          />
        </div>
        <div>
          <label className={labelClass} htmlFor="ai-llm-maxtokens">
            Max tokens
          </label>
          <input
            id="ai-llm-maxtokens"
            data-cy="ai-chat-maxtokens-input"
            type="number"
            min={1}
            step={1}
            className={fieldClass}
            value={draft.maxTokens}
            onChange={(e) =>
              update({ maxTokens: Number(e.target.value) || 0 })
            }
          />
        </div>
      </div>

      <div className="flex justify-end">
        <Button
          color="primary"
          size="small"
          data-cy="ai-chat-save-settings"
          onClick={handleSave}
        >
          Save
        </Button>
      </div>
    </div>
  );
}
