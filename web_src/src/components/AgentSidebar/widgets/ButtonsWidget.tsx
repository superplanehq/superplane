import { Button } from "@/components/ui/button";

interface ButtonsWidgetProps {
  prompt: string;
  items: string[];
  onAction?: (text: string) => void;
}

export function ButtonsWidget({ prompt, items, onAction }: ButtonsWidgetProps) {
  return (
    <div className="my-4 rounded-lg border border-slate-200 bg-white overflow-hidden">
      {prompt && (
        <div className="px-3 py-2 bg-slate-50 border-b border-slate-200">
          <p className="text-xs font-medium text-slate-900">{prompt}</p>
        </div>
      )}
      <div className="p-2 flex flex-col gap-1.5 overflow-x-auto">
        {items.map((item, i) => (
          <Button
            key={item}
            variant="ghost"
            size="sm"
            className="justify-start text-xs text-slate-700 hover:bg-slate-50 hover:text-slate-900 h-auto py-2 px-3 text-left whitespace-normal"
            onClick={() => onAction?.(item)}
          >
            <span className="inline-flex items-center justify-center size-5 rounded bg-slate-100 text-slate-700 text-[10px] font-semibold mr-2 shrink-0">
              {String.fromCharCode(65 + i)}
            </span>
            {item}
          </Button>
        ))}
      </div>
    </div>
  );
}
