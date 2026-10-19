import { Blocks, type LucideIcon } from "lucide-react";

import { cn } from "@/lib/utils";

export type FactoryAppCanvasWorkspaceTogglesProps = {
  componentsOpen: boolean;
  onComponentsOpenChange: (open: boolean) => void;
};

export function FactoryAppCanvasWorkspaceToggles({
  componentsOpen,
  onComponentsOpenChange,
}: FactoryAppCanvasWorkspaceTogglesProps) {
  return (
    <div
      className="inline-flex h-7 items-center rounded-full bg-muted p-0.5"
      role="toolbar"
      aria-label="Edit workspace"
      data-testid="factory-app-workspace-toggles"
    >
      <WorkspaceToggle
        testId="factory-app-workspace-components"
        pressed={componentsOpen}
        icon={Blocks}
        label="Components"
        onPressedChange={onComponentsOpenChange}
      />
    </div>
  );
}

function WorkspaceToggle({
  testId,
  pressed,
  icon: Icon,
  label,
  onPressedChange,
}: {
  testId: string;
  pressed: boolean;
  icon: LucideIcon;
  label: string;
  onPressedChange: (open: boolean) => void;
}) {
  return (
    <button
      type="button"
      aria-pressed={pressed}
      data-testid={testId}
      onClick={() => onPressedChange(!pressed)}
      className={cn(
        "inline-flex h-6 items-center gap-1 rounded-full px-2.5 text-[12px] font-medium transition-colors",
        pressed
          ? "bg-background text-foreground shadow-sm"
          : "text-muted-foreground hover:bg-muted hover:text-foreground",
      )}
    >
      <Icon className="size-3.5" aria-hidden />
      {label}
    </button>
  );
}
