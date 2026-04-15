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
      <TabsList className="h-8 w-fit gap-0">
        <TabsTrigger value="version-edit" data-testid="canvas-view-mode-editor" aria-label="Editor">
          Editor
        </TabsTrigger>
        <TabsTrigger value="version-live" data-testid="canvas-view-mode-live" aria-label="Live Canvas">
          Live Canvas
        </TabsTrigger>
      </TabsList>
    </Tabs>
  );
}
