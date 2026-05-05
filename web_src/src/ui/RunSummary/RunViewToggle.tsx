import { useEffect } from "react";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useHeaderActionSlotSetter } from "@/ui/CanvasPage/HeaderActionSlotContext";

//
// Tab values stay `"summary"` / `"canvas"` because they're persisted in
// localStorage (`run-view-mode`) and consumed elsewhere by the page; only
// the user-facing labels are Report / Steps.
//
export type RunViewMode = "summary" | "canvas";

function ViewTabs({ value, onChange }: { value: RunViewMode; onChange: (v: RunViewMode) => void }) {
  return (
    <Tabs
      value={value}
      onValueChange={(v) => {
        if (v === "summary" || v === "canvas") onChange(v);
      }}
      className="inline-flex w-auto"
      aria-label="Run view"
    >
      <TabsList className="h-7 w-fit gap-0 rounded-sm border border-slate-300 bg-white p-0">
        <TabsTrigger
          value="summary"
          aria-label="Report"
          className="rounded-sm rounded-br-none rounded-tr-none border-none px-3 py-0.5 text-xs text-slate-600 transition-colors data-[state=active]:bg-primary data-[state=active]:text-primary-foreground data-[state=active]:shadow-none"
        >
          Report
        </TabsTrigger>
        <div className="h-full w-px bg-slate-300" />
        <TabsTrigger
          value="canvas"
          aria-label="Steps"
          className="rounded-sm rounded-bl-none rounded-tl-none border-none px-3 py-0.5 text-xs text-slate-600 transition-colors data-[state=active]:bg-primary data-[state=active]:text-primary-foreground data-[state=active]:shadow-none"
        >
          Steps
        </TabsTrigger>
      </TabsList>
    </Tabs>
  );
}

interface RunViewToggleProps {
  value: RunViewMode;
  onChange: (mode: RunViewMode) => void;
}

//
// Headless component: registers the Report / Steps tabs into the secondary
// header's action slot (the same place as Apps' "Add panel" and Live's
// "Edit") and renders nothing of its own. Mount it whenever the runs view is
// active; the cleanup clears the slot on unmount.
//
export function RunViewToggle({ value, onChange }: RunViewToggleProps) {
  const setHeaderActionNode = useHeaderActionSlotSetter();
  useEffect(() => {
    if (!setHeaderActionNode) return;
    setHeaderActionNode(<ViewTabs value={value} onChange={onChange} />);
    return () => {
      setHeaderActionNode(null);
    };
  }, [setHeaderActionNode, value, onChange]);

  return null;
}
