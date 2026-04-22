import { cn } from "@/lib/utils";

export type RunViewMode = "summary" | "canvas";

interface RunViewToggleProps {
  value: RunViewMode;
  onChange: (next: RunViewMode) => void;
  className?: string;
}

const OPTIONS: Array<{ id: RunViewMode; label: string }> = [
  { id: "summary", label: "Summary" },
  { id: "canvas", label: "Canvas" },
];

export function RunViewToggle({ value, onChange, className }: RunViewToggleProps) {
  return (
    <div
      role="tablist"
      aria-label="Run view mode"
      className={cn(
        "inline-flex items-center gap-0.5 rounded-md border border-slate-200 bg-white p-0.5 text-xs shadow-sm",
        className,
      )}
    >
      {OPTIONS.map((opt) => {
        const isActive = value === opt.id;
        return (
          <button
            key={opt.id}
            type="button"
            role="tab"
            aria-selected={isActive}
            onClick={() => onChange(opt.id)}
            className={cn(
              "rounded px-2.5 py-1 font-medium transition-colors",
              isActive ? "bg-slate-900 text-white" : "text-gray-600 hover:bg-slate-100 hover:text-gray-800",
            )}
          >
            {opt.label}
          </button>
        );
      })}
    </div>
  );
}
