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
      {/*
        Match the size of the adjacent Edit / Add panel buttons (Button
        `size="sm"`: h-7, rounded-md, text-[13px]). overflow-hidden on the
        wrapper lets the active tab's primary background paint up to the
        wrapper border without a 1px gap; h-full on the trigger overrides
        the shadcn default h-[calc(100%-1px)].
      */}
      <TabsList className="h-7 w-fit gap-0 overflow-hidden rounded-md border border-slate-300 bg-white p-0">
        <TabsTrigger
          value="summary"
          aria-label="Report"
          className="h-full rounded-l-md rounded-r-none border-none px-3 py-0 text-[13px] text-slate-600 transition-colors data-[state=active]:bg-primary data-[state=active]:text-primary-foreground data-[state=active]:shadow-none"
        >
          Report
        </TabsTrigger>
        <div className="h-full w-px bg-slate-300" />
        <TabsTrigger
          value="canvas"
          aria-label="Steps"
          className="h-full rounded-l-none rounded-r-md border-none px-3 py-0 text-[13px] text-slate-600 transition-colors data-[state=active]:bg-primary data-[state=active]:text-primary-foreground data-[state=active]:shadow-none"
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
