import type { ComponentType, ReactNode } from "react";
import { FileText } from "lucide-react";
import type { NodeChipContext } from "@/ui/Markdown/CanvasMarkdown";
import { MarkdownPanel, type MarkdownPanelContent } from "./MarkdownPanel";

/**
 * Context shared with every panel renderer. Today this is just the node-ref
 * data needed by Markdown panels (so `@node` chips render rich previews); it
 * lives here so future panel types (charts, run lists, etc.) can pick what
 * they need without their renderers receiving panel-type-specific props.
 */
export interface PanelRenderCtx {
  nodeRefs?: NodeChipContext;
}

export interface PanelRenderProps<TContent> {
  content: TContent;
  readOnly: boolean;
  onChange: (next: TContent) => void;
  ctx: PanelRenderCtx;
}

// PanelContent is whatever JSON payload a panel type chooses to persist as
// its `content` field. We carry it as `unknown`-ish here so each panel def
// can narrow it to its own shape via `normalize`.
export type PanelContent = Record<string, unknown>;

export interface PanelDef<TContent = PanelContent> {
  type: string;
  label: string;
  icon: ComponentType<{ className?: string }>;
  defaultContent: TContent;
  defaultSize: { w: number; h: number; minW?: number; minH?: number };
  /**
   * Best-effort coercion of a stored panel.content into the typed shape this
   * renderer expects. Used when reading from the API; missing/malformed fields
   * should be normalized to safe defaults rather than throwing.
   */
  normalize: (raw: PanelContent | undefined) => TContent;
  render: (props: PanelRenderProps<TContent>) => ReactNode;
}

const markdownPanelDef: PanelDef<MarkdownPanelContent> = {
  type: "markdown",
  label: "Markdown",
  icon: FileText,
  defaultContent: { body: "" },
  defaultSize: { w: 6, h: 6, minW: 2, minH: 2 },
  normalize: (raw) => ({
    body: typeof raw?.body === "string" ? (raw.body as string) : "",
  }),
  render: (props) => <MarkdownPanel {...props} />,
};

const registry: Record<string, PanelDef> = {
  [markdownPanelDef.type]: markdownPanelDef as unknown as PanelDef,
};

export const panelRegistry: Readonly<Record<string, PanelDef>> = registry;

export function getPanelDef(type: string): PanelDef | undefined {
  return registry[type];
}

export function listPanelDefs(): PanelDef[] {
  return Object.values(registry);
}
