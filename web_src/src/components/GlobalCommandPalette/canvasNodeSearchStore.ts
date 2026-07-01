import { useSyncExternalStore } from "react";
import { resolveIcon } from "@/lib/utils";
import type { PaletteAction } from "./types";

export type CanvasNodeSearchResult = {
  id: string;
  label: string;
  iconSlug: string;
  keywords: string[];
};

export type CanvasNodeSearchProvider = {
  searchNodes: (query: string) => CanvasNodeSearchResult[];
  selectNode: (nodeId: string) => void;
};

let activeProvider: CanvasNodeSearchProvider | null = null;
const listeners = new Set<() => void>();

export function registerCanvasNodeSearchProvider(provider: CanvasNodeSearchProvider) {
  activeProvider = provider;
  emitCanvasNodeSearchChange();

  return () => {
    if (activeProvider === provider) {
      activeProvider = null;
      emitCanvasNodeSearchChange();
    }
  };
}

export function useCanvasNodeSearchProvider() {
  return useSyncExternalStore(subscribeToCanvasNodeSearch, getCanvasNodeSearchSnapshot, getCanvasNodeSearchSnapshot);
}

export function buildCanvasNodeSearchActions({
  closePalette,
  provider,
  query,
}: {
  closePalette: () => void;
  provider: CanvasNodeSearchProvider | null;
  query: string;
}): PaletteAction[] {
  if (!provider) return [];

  const normalizedQuery = query.trim();
  if (!normalizedQuery) return [];

  return provider.searchNodes(normalizedQuery).map((node) => ({
    id: node.id,
    label: node.label,
    description: node.id,
    icon: resolveIcon(node.iconSlug),
    keywords: node.keywords,
    onSelect: () => {
      provider.selectNode(node.id);
      closePalette();
    },
  }));
}

function subscribeToCanvasNodeSearch(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function getCanvasNodeSearchSnapshot() {
  return activeProvider;
}

function emitCanvasNodeSearchChange() {
  listeners.forEach((listener) => listener());
}
