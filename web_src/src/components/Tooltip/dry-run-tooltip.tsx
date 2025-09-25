import Tippy from '@tippyjs/react';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import 'tippy.js/dist/tippy.css';

interface DryRunTooltipProps {
  className?: string;
}

export function DryRunTooltip({ className = '' }: DryRunTooltipProps) {
  return (
    <Tippy
      content="When enabled, this stage will use the no-op executor regardless of the configured executor type"
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