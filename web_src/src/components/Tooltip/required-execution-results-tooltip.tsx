import Tippy from '@tippyjs/react';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import 'tippy.js/dist/tippy.css';

interface RequiredExecutionResultsTooltipProps {
  className?: string;
}

export function RequiredExecutionResultsTooltip({ className = '' }: RequiredExecutionResultsTooltipProps) {
  return (
    <Tippy
      content="Set the input's value to what it was in the last execution. You can specify whether to use the value from a passed, failed, or any previous execution."
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