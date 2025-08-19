import Tippy from '@tippyjs/react/headless';
import { ReactElement } from 'react';

interface IOTooltipProps {
  type: 'inputs' | 'outputs';
  data: Array<{ name: string | undefined; value: string | undefined }>;
  children: React.ReactNode;
}

export function IOTooltip({ type, data, children }: IOTooltipProps) {
  const title = type === 'inputs' ? 'Inputs' : 'Outputs';

  return (
    <Tippy
      render={() => (
        <div className="min-w-[250px] max-w-xs">
          <div className="bg-white dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 rounded-lg p-3 shadow-lg">
            <div className="flex items-start gap-3">
              <div className="flex-1">
                <div className="w-full text-left text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">{title}</div>
                <div className="space-y-1">
                  {data?.map((item, index) => (
                    <div key={index} className="flex items-center justify-between">
                      <span className="text-xs text-gray-600 dark:text-zinc-300 font-medium">{item.name || 'Unknown'}</span>
                      <div className="flex items-center gap-2">
                        <span className="font-mono !text-xs inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-zinc-600/10 text-zinc-700 group-data-hover:bg-zinc-600/20 dark:bg-white/5 dark:text-zinc-400 dark:group-data-hover:bg-white/10">{item.value || 'N/A'}</span>
                      </div>
                    </div>
                  ))}
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
  );
}