import type { LucideIcon } from "lucide-react";
import { Folder, List, Plug, Sparkles, Wrench } from "lucide-react";

export const MARKDOWN_SECTION_PRESET_IDS = ["tools", "rules", "skills", "mcp", "folder"] as const;

export type MarkdownSectionPresetId = (typeof MARKDOWN_SECTION_PRESET_IDS)[number];

export type MarkdownSectionPreset = {
  id: MarkdownSectionPresetId;
  Icon: LucideIcon;
  /** Icon accent */
  iconClassName: string;
  /** Tinted header bar for root-level sections */
  barClassName: string;
};

/**
 * Named presets authors can pick with `> [!SECTION:tools] Title`.
 * Colors follow the Context Usage category accents (purple tools, green rules, …).
 */
export const MARKDOWN_SECTION_PRESETS: Record<MarkdownSectionPresetId, MarkdownSectionPreset> = {
  tools: {
    id: "tools",
    Icon: Wrench,
    iconClassName: "text-violet-600 dark:text-violet-400",
    barClassName: "bg-violet-100/80 dark:bg-violet-950/50",
  },
  rules: {
    id: "rules",
    Icon: List,
    iconClassName: "text-emerald-600 dark:text-emerald-400",
    barClassName: "bg-emerald-100/70 dark:bg-emerald-950/40",
  },
  skills: {
    id: "skills",
    Icon: Sparkles,
    iconClassName: "text-cyan-600 dark:text-cyan-400",
    barClassName: "bg-cyan-100/70 dark:bg-cyan-950/40",
  },
  mcp: {
    id: "mcp",
    Icon: Plug,
    iconClassName: "text-sky-600 dark:text-sky-400",
    barClassName: "bg-sky-100/70 dark:bg-sky-950/40",
  },
  folder: {
    id: "folder",
    Icon: Folder,
    iconClassName: "text-slate-500 dark:text-gray-400",
    barClassName: "bg-slate-100/90 dark:bg-gray-800/80",
  },
};

const DEFAULT_ROOT_PRESET = MARKDOWN_SECTION_PRESETS.rules;
const DEFAULT_NESTED_PRESET = MARKDOWN_SECTION_PRESETS.folder;

export function isMarkdownSectionPresetId(value: string): value is MarkdownSectionPresetId {
  return (MARKDOWN_SECTION_PRESET_IDS as readonly string[]).includes(value);
}

export function resolveMarkdownSectionPreset(presetId: string | undefined, depth: number): MarkdownSectionPreset {
  if (presetId && isMarkdownSectionPresetId(presetId)) {
    return MARKDOWN_SECTION_PRESETS[presetId];
  }

  return depth === 0 ? DEFAULT_ROOT_PRESET : DEFAULT_NESTED_PRESET;
}
