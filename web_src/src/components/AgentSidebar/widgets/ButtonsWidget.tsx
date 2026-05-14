import { Button } from "@/components/ui/button";

interface ButtonsWidgetProps {
  prompt: string;
  items: string[];
  onAction?: (text: string) => void;
}

export function ButtonsWidget({ prompt, items, onAction }: ButtonsWidgetProps) {
  return (
    <div className="my-2 rounded-lg border border-violet-200 overflow-hidden">
      {prompt && (
        <div className="px-3 py-2 bg-violet-50 border-b border-violet-200">
          <p className="text-xs font-medium text-violet-900">{prompt}</p>
        </div>
      )}
      <div className="p-2 flex flex-col gap-1.5">
        {items.map((item, i) => (
          <Button
            key={item}
            variant="ghost"
            size="sm"
            className="justify-start text-xs text-slate-700 hover:bg-violet-50 hover:text-violet-900 h-auto py-2 px-3"
            onClick={() => onAction?.(item)}
          >
            <span className="inline-flex items-center justify-center size-5 rounded bg-violet-100 text-violet-700 text-[10px] font-semibold mr-2 shrink-0">
              {String.fromCharCode(65 + i)}
            </span>
            {item}
          </Button>
        ))}
      </div>
    </div>
  );
}
