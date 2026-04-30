import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";

type CanvasMode = "version-live" | "version-edit";

interface CanvasModeToggleProps {
  mode: CanvasMode;
  onSelectEditor: () => void;
  onSelectLive: () => void;
}

export function CanvasModeToggle({ mode, onSelectEditor, onSelectLive }: CanvasModeToggleProps) {
  const handleValueChange = (next: string) => {
    if (next === "version-edit" && mode === "version-live") {
      void onSelectEditor();
    } else if (next === "version-live" && mode === "version-edit") {
      void onSelectLive();
    }
  };

  return (
    <Tabs value={mode} onValueChange={handleValueChange} className="inline-flex w-auto" aria-label="Canvas view">
      <TabsList className="h-8 w-fit gap-0 rounded-sm border border-slate-300 bg-white/80 p-0">
        <TabsTrigger
          value="version-edit"
          data-testid="canvas-view-mode-editor"
          aria-label="Editor"
          className="rounded-sm rounded-br-none rounded-tr-none border-none px-3 py-1 text-slate-600 transition-colors data-[state=active]:bg-sky-50 data-[state=active]:text-sky-700 data-[state=active]:shadow-none"
        >
          Editor
        </TabsTrigger>
        <div className="h-full w-px bg-slate-300"></div>
        <TabsTrigger
          value="version-live"
          data-testid="canvas-view-mode-live"
          aria-label="Live Canvas"
          className="rounded-sm rounded-bl-none rounded-tl-none border-none px-3 py-1 text-slate-600 transition-colors data-[state=active]:bg-sky-50 data-[state=active]:text-sky-700 data-[state=active]:shadow-none"
        >
          Live Canvas
        </TabsTrigger>
      </TabsList>
    </Tabs>
  );
}
