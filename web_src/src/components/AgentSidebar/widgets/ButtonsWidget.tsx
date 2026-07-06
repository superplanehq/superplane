import { Button } from "@/components/ui/button";

interface ButtonsWidgetProps {
  prompt: string;
  items: string[];
  onAction?: (text: string) => void;
}

export function ButtonsWidget({ prompt, items, onAction }: ButtonsWidgetProps) {
  return (
    <div className="my-4 overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-gray-700 dark:bg-gray-800">
      {prompt && (
        <div className="border-b border-slate-200 bg-slate-50 px-3 py-2 dark:border-gray-700 dark:bg-gray-900/60">
          <p className="text-xs font-medium text-slate-900 dark:text-gray-100">{prompt}</p>
        </div>
      )}
      <div className="flex flex-col gap-1.5 overflow-x-auto p-2">
        {items.map((item, i) => (
          <Button
            key={item}
            variant="ghost"
            size="sm"
            className="h-auto justify-start whitespace-normal px-3 py-2 text-left text-xs text-slate-700 hover:bg-slate-50 hover:text-slate-900 dark:text-gray-300 dark:hover:bg-gray-700 dark:hover:text-gray-100"
            onClick={() => onAction?.(item)}
          >
            <span className="mr-2 inline-flex size-5 shrink-0 items-center justify-center rounded bg-slate-100 text-[10px] font-semibold text-slate-700 dark:bg-gray-700 dark:text-gray-200">
              {String.fromCharCode(65 + i)}
            </span>
            {item}
          </Button>
        ))}
      </div>
    </div>
  );
}
