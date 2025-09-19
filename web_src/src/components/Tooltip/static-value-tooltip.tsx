import Tippy from '@tippyjs/react/headless';
import { ReactElement } from 'react';

interface StaticValueTooltipProps {
  children: React.ReactNode;
}

export function StaticValueTooltip({ children }: StaticValueTooltipProps) {
  return (
    <div className="flex items-center gap-2">
      <Tippy
        render={() => (
          <div className="min-w-[300px] max-w-sm">
            <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-4 text-sm z-50">
              <div className="font-semibold mb-3 text-zinc-900 dark:text-zinc-100">Static Value - tooltip</div>
            </div>
          </div>
        )}
        placement="top"
        interactive={true}
        delay={200}
      >
        {children as ReactElement}
      </Tippy>
    </div>
  );
}