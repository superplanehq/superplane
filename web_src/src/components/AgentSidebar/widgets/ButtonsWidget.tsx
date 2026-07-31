import { Button } from "@/components/ui/button";

interface ButtonsWidgetProps {
  prompt: string;
  items: string[];
  onAction?: (text: string) => void;
}

export function ButtonsWidget({ prompt, items, onAction }: ButtonsWidgetProps) {
  return (
    <div className="my-4 overflow-hidden rounded-lg border border-edge-default bg-surface-raised">
      {prompt && (
        <div className="border-b border-edge-default bg-surface-subtle px-3 py-2">
          <p className="text-xs font-medium text-content-primary">{prompt}</p>
        </div>
      )}
      <div className="flex flex-col gap-1.5 overflow-x-auto p-2">
        {items.map((item, i) => (
          <Button
            key={item}
            variant="ghost"
            size="sm"
            className="h-auto justify-start whitespace-normal px-3 py-2 text-left text-xs text-content-secondary hover:bg-action-neutral-hover hover:text-content-primary"
            onClick={() => onAction?.(item)}
          >
            <span className="mr-2 inline-flex size-5 shrink-0 items-center justify-center rounded bg-action-neutral text-[10px] font-semibold text-content-secondary">
              {String.fromCharCode(65 + i)}
            </span>
            {item}
          </Button>
        ))}
      </div>
    </div>
  );
}
