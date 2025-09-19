import Tippy from '@tippyjs/react';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import 'tippy.js/dist/tippy.css';

interface RefTooltipProps {
  className?: string;
}

export function RefTooltip({ className = '' }: RefTooltipProps) {
  return (
    <Tippy
      content="The Git ref (branch, tag, or pull request) where the executor will find the workflow or pipeline YAML file to run."
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