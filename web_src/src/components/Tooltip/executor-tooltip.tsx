import Tippy from '@tippyjs/react';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import 'tippy.js/dist/tippy.css';

interface ExecutorTooltipProps {
  className?: string;
}

export function ExecutorTooltip({ className = '' }: ExecutorTooltipProps) {
  return (
    <Tippy
      content="Configure what action this component will perform - specify pipelines, API endpoints, parameters, and execution settings."
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