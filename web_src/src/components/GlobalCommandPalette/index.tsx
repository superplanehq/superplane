import { CommandDialog, CommandEmpty, CommandInput, CommandList } from "@/components/ui/command";
import { useCommandPaletteModel } from "./model";
import type { CommandPaletteModel } from "./model";
import {
  AdminCommandPage,
  CanvasNodeSearchPage,
  CanvasSettingsPage,
  OpenCanvasPage,
  OrganizationSettingsPage,
  RootCommandPage,
} from "./pages";
import { pageTitle } from "./route";

export function GlobalCommandPalette() {
  const model = useCommandPaletteModel();
  if (!model) return null;
  return <CommandPaletteDialog model={model} />;
}

function CommandPaletteDialog({ model }: { model: CommandPaletteModel }) {
  return (
    <CommandDialog
      open={model.open}
      onOpenChange={model.setOpen}
      title="Command Palette"
      description="Search pages, actions, and utilities."
      className="top-[12vh] max-h-[min(760px,80vh)] w-[calc(100vw-2rem)] max-w-3xl translate-y-0 overflow-hidden rounded-xl border border-slate-200 bg-white p-0 shadow-2xl sm:top-[14vh]"
      showCloseButton={false}
    >
      <CommandInput
        value={model.search}
        onValueChange={model.setSearch}
        placeholder={model.page === "root" ? "What can we help with?" : pageTitle(model.page)}
        className="h-16 text-lg"
      />
      <CommandList className="max-h-[min(600px,calc(80vh-4rem))] scroll-py-2 px-3 py-3">
        <CommandEmpty>No commands found.</CommandEmpty>
        <PalettePageContent model={model} />
      </CommandList>
    </CommandDialog>
  );
}

function PalettePageContent({ model }: { model: CommandPaletteModel }) {
  const onBack = () => {
    model.setSearch("");
    model.setPage("root");
  };
  const onOpenPage = (page: Parameters<typeof model.setPage>[0]) => {
    model.setSearch("");
    model.setPage(page);
  };

  switch (model.page) {
    case "organization-settings":
      return (
        <OrganizationSettingsPage
          actions={model.settingsActions}
          onBack={onBack}
          organizationName={model.organizationName}
        />
      );
    case "canvas-settings":
      return (
        <CanvasSettingsPage
          {...model.canvasListProps}
          canvasId={model.canvasId}
          currentCanvasName={model.currentCanvasName}
          onBack={onBack}
        />
      );
    case "open-canvas":
      return <OpenCanvasPage {...model.canvasListProps} onBack={onBack} />;
    case "node-search":
      return <CanvasNodeSearchPage actions={model.canvasNodeSearchActions} onBack={onBack} />;
    case "admin":
      return <AdminCommandPage actions={model.adminActions} onBack={onBack} />;
    default:
      return (
        <RootCommandPage
          currentCanvasActions={model.currentCanvasActions}
          onOpenPage={onOpenPage}
          rootActions={model.rootActions}
          rootPageActions={model.rootPageActions}
        />
      );
  }
}
