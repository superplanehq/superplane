import { CommandDialog, CommandEmpty, CommandInput, CommandList } from "@/components/ui/command";
import { useCommandPaletteModel } from "./model";
import type { CommandPaletteModel } from "./model";
import { CommandPalettePage } from "./pages";
import { useCommandPalettePageProps } from "./usePageProps";

export function GlobalCommandPalette() {
  const model = useCommandPaletteModel();
  if (!model) return null;
  return <CommandPaletteDialog model={model} />;
}

function CommandPaletteDialog({ model }: { model: CommandPaletteModel }) {
  const pageProps = useCommandPalettePageProps(model);

  return (
    <CommandDialog
      open={model.open}
      onOpenChange={pageProps.handleOpenChange}
      title="Command Palette"
      description="Search apps, settings, and commands."
      className="top-[12vh] max-h-[min(760px,80vh)] w-[calc(100vw-2rem)] max-w-3xl translate-y-0 overflow-hidden rounded-xl border border-slate-200 bg-white p-0 shadow-2xl sm:top-[14vh]"
      showCloseButton={false}
    >
      <CommandInput
        value={model.search}
        onValueChange={pageProps.handleSetSearch}
        placeholder="Find apps, integrations, and commands..."
        className="h-16 text-lg"
      />
      <CommandList className="max-h-[min(600px,calc(80vh-4rem))] scroll-py-2 px-3 py-3">
        <CommandEmpty>No results found.</CommandEmpty>
        <CommandPalettePage {...pageProps} />
      </CommandList>
    </CommandDialog>
  );
}
