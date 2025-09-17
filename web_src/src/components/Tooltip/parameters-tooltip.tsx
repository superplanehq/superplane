import Tippy from '@tippyjs/react';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import 'tippy.js/dist/tippy.css';

interface ParametersTooltipProps {
  executorType: string;
  className?: string;
}

// Map of executor types to their parameter descriptions
const PARAMETER_DESCRIPTIONS: Record<string, string> = {
  semaphore: 'Parameters are values that will be forwarded to your Semaphore pipeline and available as environment variables.',
  github: 'Inputs are values that will be forwarded to your GitHub workflow and available as workflow inputs.',
  http: 'These values will be forwarded to your HTTP endpoint as request parameters.'
};

export function ParametersTooltip({ executorType, className = '' }: ParametersTooltipProps) {
  const description = PARAMETER_DESCRIPTIONS[executorType] || 'These values will be forwarded to your pipeline execution.';

  return (
    <Tippy
      content={description}
      placement="top"
      arrow={true}
      theme="dark"
      maxWidth={300}
    >
      <div
        className={`text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300 transition-colors cursor-help ${className}`}
        role="button"
        tabIndex={0}
      >
        <MaterialSymbol name="help" size="sm" />
      </div>
    </Tippy>
  );
}