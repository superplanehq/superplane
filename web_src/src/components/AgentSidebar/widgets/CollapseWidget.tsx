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
    <div className="my-4 border border-slate-200 rounded-lg overflow-hidden">
      <button
        type="button"
        onClick={() => setOpen((prev) => !prev)}
        className="w-full flex items-center gap-2 px-3 py-2 text-left text-xs font-medium text-slate-700 hover:bg-slate-50 cursor-pointer"
      >
        <ChevronRight className={cn("size-3.5 transition-transform", open && "rotate-90")} />
        {title}
      </button>
      {open && (
        <div className="px-3 pb-2 border-t border-slate-100">
          <pre className="text-xs text-slate-600 whitespace-pre-wrap break-words mt-2 font-mono">{content}</pre>
        </div>
      )}
    </div>
  );
}
