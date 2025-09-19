import Tippy from '@tippyjs/react';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import 'tippy.js/dist/tippy.css';

interface ExecutionNameTooltipProps {
  className?: string;
}

export function ExecutionNameTooltip({ className = '' }: ExecutionNameTooltipProps) {
  return (
    <Tippy
      content="Configure a display name for all executions within this stage. You can use stage inputs as variables and combine them with static text. If not set, the Execution ID will be used as the display name."
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