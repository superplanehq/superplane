import { CommandGroup, CommandSeparator } from "@/components/ui/command";
import { Palette, Settings } from "lucide-react";
import { openPageAction } from "./actions";
import { ActionItem, CanvasListItems, NestedPage, PageItem } from "./items";
import type { CanvasCommandListProps, CommandPage, PaletteAction, PalettePageAction } from "./types";

export function RootCommandPage({
  currentCanvasActions,
  onOpenPage,
  rootActions,
  rootPageActions,
}: {
  currentCanvasActions: PaletteAction[];
  onOpenPage: (page: CommandPage) => void;
  rootActions: PaletteAction[];
  rootPageActions: PalettePageAction[];
}) {
  return (
    <>
      <CommandGroup heading="Create">
        {rootActions.slice(0, 2).map((action) => (
          <ActionItem key={action.id} action={action} />
        ))}
      </CommandGroup>

      <CommandSeparator className="my-2" />

      {currentCanvasActions.length > 0 ? (
        <>
          <CommandGroup heading="Current Canvas">
            {currentCanvasActions.map((action) => (
              <ActionItem key={action.id} action={action} />
            ))}
          </CommandGroup>
          <CommandSeparator className="my-2" />
        </>
      ) : null}

      <CommandGroup heading="Navigate">
        {rootPageActions.map((action) => (
          <PageItem key={action.id} action={action} onSelect={openPageAction(action.page, onOpenPage)} />
        ))}
        {rootActions.slice(2, -2).map((action) => (
          <ActionItem key={action.id} action={action} />
        ))}
      </CommandGroup>

      <CommandSeparator className="my-2" />

      <CommandGroup heading="Help and Account">
        {rootActions.slice(-2).map((action) => (
          <ActionItem key={action.id} action={action} />
        ))}
      </CommandGroup>
    </>
  );
}

export function OrganizationSettingsPage({
  actions,
  onBack,
  organizationName,
}: {
  actions: PaletteAction[];
  onBack: () => void;
  organizationName: string;
}) {
  return (
    <NestedPage onBack={onBack}>
      <CommandGroup heading={organizationName}>
        {actions.map((action) => (
          <ActionItem key={action.id} action={action} />
        ))}
      </CommandGroup>
    </NestedPage>
  );
}

export function CanvasSettingsPage({
  canvasId,
  currentCanvasName,
  onBack,
  ...canvasListProps
}: CanvasCommandListProps & {
  canvasId: string | null;
  currentCanvasName: string;
  onBack: () => void;
}) {
  const { goTo, organizationId } = canvasListProps;

  return (
    <NestedPage onBack={onBack}>
      <CommandGroup heading={canvasId ? "Current Canvas" : "Canvases"}>
        {canvasId ? (
          <ActionItem
            action={{
              id: "current-canvas-settings",
              label: currentCanvasName,
              description: "Open canvas settings",
              icon: Settings,
              onSelect: () => organizationId && goTo(`/${organizationId}/canvases/${canvasId}/settings`),
            }}
          />
        ) : null}
        <CanvasSettingsList {...canvasListProps} />
      </CommandGroup>
    </NestedPage>
  );
}

export function OpenCanvasPage({ onBack, ...canvasListProps }: CanvasCommandListProps & { onBack: () => void }) {
  return (
    <NestedPage onBack={onBack}>
      <CommandGroup heading="Canvases">
        <OpenCanvasList {...canvasListProps} />
      </CommandGroup>
    </NestedPage>
  );
}

export function AdminCommandPage({ actions, onBack }: { actions: PaletteAction[]; onBack: () => void }) {
  return (
    <NestedPage onBack={onBack}>
      <CommandGroup heading="Installation Admin">
        {actions.map((action) => (
          <ActionItem key={action.id} action={action} />
        ))}
      </CommandGroup>
    </NestedPage>
  );
}

function CanvasSettingsList({ goTo, organizationId, ...props }: CanvasCommandListProps) {
  return (
    <CanvasListItems
      {...props}
      description="Open canvas settings"
      emptyLabel="No canvases available."
      icon={Settings}
      onSelect={(canvas) => {
        const id = canvas.metadata?.id;
        if (organizationId && id) goTo(`/${organizationId}/canvases/${id}/settings`);
      }}
    />
  );
}

function OpenCanvasList({ goTo, organizationId, ...props }: CanvasCommandListProps) {
  return (
    <CanvasListItems
      {...props}
      description="Open canvas"
      emptyLabel="No canvases available."
      icon={Palette}
      onSelect={(canvas) => {
        const id = canvas.metadata?.id;
        if (organizationId && id) goTo(`/${organizationId}/canvases/${id}`);
      }}
    />
  );
}
