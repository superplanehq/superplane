import Tippy from '@tippyjs/react/headless';
import { ReactElement } from 'react';

interface ExpressionTooltipProps {
  children: React.ReactNode;
}

export function ExpressionTooltip({ children }: ExpressionTooltipProps) {
  return (
    <div className="flex items-center gap-2">
      <Tippy
        render={() => (
          <div className="min-w-[300px] max-w-sm">
            <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-4 text-sm z-50">
              <div className="font-semibold mb-3 text-zinc-900 dark:text-zinc-100">Using Expressions to Map Event Data</div>
              <p className="text-xs text-zinc-600 dark:text-zinc-400 mb-3">
                Use a JSONPath expression to extract a value from the incoming event's payload and map it to this input. Here are some examples:
              </p>
              <div className="space-y-3">
                <div>
                  <div className="text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-2">Examples of Expressions:</div>
                  <div className="space-y-2 ml-2">
                    <div>
                      <div className="text-xs text-zinc-600 dark:text-zinc-400 mb-1">Select a top-level property:</div>
                      <code className="text-xs bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded text-zinc-800 dark:text-zinc-200 font-mono">
                        $.repository.full_name
                      </code>
                    </div>
                    <div>
                      <div className="text-xs text-zinc-600 dark:text-zinc-400 mb-1">Conditional selection:</div>
                      <code className="text-xs bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded text-zinc-800 dark:text-zinc-200 font-mono">
                        $.pull_request?.head?.ref
                      </code>
                    </div>
                    <div>
                      <div className="text-xs text-zinc-600 dark:text-zinc-400 mb-1">Select an item from an array:</div>
                      <code className="text-xs bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded text-zinc-800 dark:text-zinc-200 font-mono">
                        $.commits[0].message
                      </code>
                    </div>
                  </div>
                </div>
              </div>
              <div className="text-xs text-zinc-500 dark:text-zinc-400 mt-4">
                Expressions are parsed using the <a href="https://expr-lang.org" target="_blank" rel="noopener noreferrer" className="text-blue-600 dark:text-blue-400 hover:underline">Expr</a> language.
              </div>
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