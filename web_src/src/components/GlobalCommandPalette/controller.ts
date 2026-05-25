import type { CommandPage } from "./types";

const OPEN_COMMAND_PALETTE_EVENT = "superplane:open-command-palette";

export type OpenCommandPaletteOptions = {
  page?: CommandPage;
  search?: string;
};

export function openGlobalCommandPalette(options: OpenCommandPaletteOptions = {}) {
  window.dispatchEvent(new CustomEvent<OpenCommandPaletteOptions>(OPEN_COMMAND_PALETTE_EVENT, { detail: options }));
}

export function subscribeToOpenCommandPalette(listener: (options: OpenCommandPaletteOptions) => void) {
  const onOpenCommandPalette = (event: Event) => {
    listener((event as CustomEvent<OpenCommandPaletteOptions>).detail ?? {});
  };

  window.addEventListener(OPEN_COMMAND_PALETTE_EVENT, onOpenCommandPalette);
  return () => window.removeEventListener(OPEN_COMMAND_PALETTE_EVENT, onOpenCommandPalette);
}
