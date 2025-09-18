import Tippy from '@tippyjs/react/headless';
import { ReactElement } from 'react';

interface FilterTooltipProps {
  children: React.ReactNode;
}

export function FilterTooltip({ children }: FilterTooltipProps) {
  return (
    <div className="flex items-center gap-2">
      <Tippy
        render={() => (
          <div className="min-w-[300px] max-w-sm">
            <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-4 text-sm z-50">
              <div className="font-semibold mb-3 text-zinc-900 dark:text-zinc-100">Filter Examples</div>
              <div className="space-y-2">
                <div className="flex items-start gap-2">
                  <span className="text-zinc-500 text-base leading-none mt-1">•</span>
                  <div>
                    <code className="bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded text-xs font-mono text-zinc-800 dark:text-zinc-200">$.type == "push"</code>
                    <div className="text-xs text-zinc-600 dark:text-zinc-400 mt-1">Match specific event type</div>
                  </div>
                </div>
                <div className="flex items-start gap-2">
                  <span className="text-zinc-500 text-base leading-none mt-1">•</span>
                  <div>
                    <code className="bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded text-xs font-mono text-zinc-800 dark:text-zinc-200">$.ref == "refs/heads/main"</code>
                    <div className="text-xs text-zinc-600 dark:text-zinc-400 mt-1">GitHub branch filter</div>
                  </div>
                </div>
                <div className="flex items-start gap-2">
                  <span className="text-zinc-500 text-base leading-none mt-1">•</span>
                  <div>
                    <code className="bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded text-xs font-mono text-zinc-800 dark:text-zinc-200">$.action in ["opened", "closed"]</code>
                    <div className="text-xs text-zinc-600 dark:text-zinc-400 mt-1">Multiple values</div>
                  </div>
                </div>
                <div className="flex items-start gap-2">
                  <span className="text-zinc-500 text-base leading-none mt-1">•</span>
                  <div>
                    <code className="bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded text-xs font-mono text-zinc-800 dark:text-zinc-200">$.payload.size {'>'}  100</code>
                    <div className="text-xs text-zinc-600 dark:text-zinc-400 mt-1">Numeric comparison</div>
                  </div>
                </div>
                <div className="flex items-start gap-2">
                  <span className="text-zinc-500 text-base leading-none mt-1">•</span>
                  <div>
                    <code className="bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded text-xs font-mono text-zinc-800 dark:text-zinc-200">has($.repository.private)</code>
                    <div className="text-xs text-zinc-600 dark:text-zinc-400 mt-1">Check field exists</div>
                  </div>
                </div>
                <div className="flex items-start gap-2">
                  <span className="text-zinc-500 text-base leading-none mt-1">•</span>
                  <div>
                    <code className="bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded text-xs font-mono text-zinc-800 dark:text-zinc-200">$.branch matches "^feature/"</code>
                    <div className="text-xs text-zinc-600 dark:text-zinc-400 mt-1">Regex pattern</div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
        placement="top"
        interactive={true}
      >
        {children as ReactElement}
      </Tippy>
    </div>
  );
}