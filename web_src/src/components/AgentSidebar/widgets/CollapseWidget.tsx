import { ChevronRight } from "lucide-react";
import { useState } from "react";
import { cn } from "@/lib/utils";

interface CollapseWidgetProps {
  title: string;
  content: string;
}

export function CollapseWidget({ title, content }: CollapseWidgetProps) {
  const [open, setOpen] = useState(false);

  return (
    <div className="my-4 overflow-hidden rounded-lg border border-slate-200 dark:border-gray-700 dark:bg-gray-800">
      <button
        type="button"
        onClick={() => setOpen((prev) => !prev)}
        className="flex w-full cursor-pointer items-center gap-2 px-3 py-2 text-left text-xs font-medium text-slate-700 hover:bg-slate-50 dark:text-gray-300 dark:hover:bg-gray-700"
      >
        <ChevronRight className={cn("size-3.5 transition-transform", open && "rotate-90")} />
        {title}
      </button>
      {open && (
        <div className="border-t border-slate-100 px-3 pb-2 dark:border-gray-700">
          <pre className="mt-2 whitespace-pre-wrap break-words font-mono text-xs text-slate-600 dark:text-gray-300">
            {content}
          </pre>
        </div>
      )}
    </div>
  );
}
