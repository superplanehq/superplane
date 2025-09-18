import Tippy from '@tippyjs/react/headless';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';

interface FiltersTooltipProps {
  className?: string;
}

export function FiltersTooltip({ className = '' }: FiltersTooltipProps) {
  const filterExamples = (
    <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-4 min-w-[450px]">
      <div className="text-sm font-medium text-zinc-900 dark:text-white mb-3">Filter Examples</div>
      <div className="space-y-3">
        <div>
          <div className="text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-2">Data Filters (JSONPath expressions):</div>
          <div className="space-y-2 ml-2">
            <div>
              <div className="text-xs text-zinc-600 dark:text-zinc-400 mb-1">Equality:</div>
              <code className="text-xs bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded text-zinc-800 dark:text-zinc-200 font-mono">
                $.ref == "refs/heads/main"
              </code>
            </div>
            <div>
              <div className="text-xs text-zinc-600 dark:text-zinc-400 mb-1">Contains matching:</div>
              <code className="text-xs bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded text-zinc-800 dark:text-zinc-200 font-mono">
                $.repository contains "api"
              </code>
            </div>
            <div>
              <div className="text-xs text-zinc-600 dark:text-zinc-400 mb-1">Nested properties:</div>
              <code className="text-xs bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded text-zinc-800 dark:text-zinc-200 font-mono">
                $.pull_request.merged == true
              </code>
            </div>
            <div>
              <div className="text-xs text-zinc-600 dark:text-zinc-400 mb-1">Array elements:</div>
              <code className="text-xs bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded text-zinc-800 dark:text-zinc-200 font-mono">
                $.commits[0].author.name == "john.doe"
              </code>
            </div>
            <div>
              <div className="text-xs text-zinc-600 dark:text-zinc-400 mb-1">Previous stage inputs:</div>
              <code className="text-xs bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded text-zinc-800 dark:text-zinc-200 font-mono">
                $.inputs.DEPLOY_URL == "https://staging.example.com"
              </code>
            </div>
          </div>
        </div>
        <div>
          <div className="text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-2">Header Filters:</div>
          <div className="space-y-2 ml-2">
            <div>
              <div className="text-xs text-zinc-600 dark:text-zinc-400 mb-1">Custom header:</div>
              <code className="text-xs bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded text-zinc-800 dark:text-zinc-200 font-mono">
                headers["x-my-header"] == "production"
              </code>
            </div>
            <div>
              <div className="text-xs text-zinc-600 dark:text-zinc-400 mb-1">User agent:</div>
              <code className="text-xs bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded text-zinc-800 dark:text-zinc-200 font-mono">
                headers["user-agent"] contains "GitHub-Hookshot"
              </code>
            </div>
          </div>
        </div>
      </div>
    </div>
  );

  return (
    <Tippy
      render={attrs => <div {...attrs}>{filterExamples}</div>}
      placement="top"
      interactive={true}
    >
      <div
        className={`text-gray-400 hover:text-gray-600 dark:text-zinc-500 dark:hover:text-zinc-300 transition-colors cursor-help ${className}`}
        role="button"
        tabIndex={0}
      >
        <MaterialSymbol name="help" size="sm" />
      </div>
    </Tippy>
  );
}