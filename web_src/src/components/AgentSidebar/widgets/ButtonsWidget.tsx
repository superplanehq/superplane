import { Button } from "@/components/ui/button";

interface ButtonsWidgetProps {
  items: string[];
  onAction?: (text: string) => void;
}

export function ButtonsWidget({ items, onAction }: ButtonsWidgetProps) {
  return (
    <div className="flex flex-wrap gap-2 my-2">
      {items.map((item) => (
        <Button
          key={item}
          variant="outline"
          size="sm"
          className="text-xs border-violet-300 hover:bg-violet-50 hover:border-violet-500"
          onClick={() => onAction?.(item)}
        >
          {item}
        </Button>
      ))}
    </div>
  );
}
