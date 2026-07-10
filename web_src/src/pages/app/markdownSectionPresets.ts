import type { LucideIcon } from "lucide-react";
import { BookOpen, Bot, Folder, LayoutDashboard, Plug, Rocket, Settings, Wrench } from "lucide-react";

export const MARKDOWN_SECTION_PRESET_IDS = [
  "overview",
  "setup",
  "runbook",
  "run",
  "troubleshoot",
  "agent",
  "integrations",
  "group",
] as const;

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
 * Named presets authors can pick with `> [!SECTION:runbook] Title`.
 * Accents are chosen for SuperPlane README / console runbook authoring.
 */
export const MARKDOWN_SECTION_PRESETS: Record<MarkdownSectionPresetId, MarkdownSectionPreset> = {
  overview: {
    id: "overview",
    Icon: LayoutDashboard,
    iconClassName: "text-slate-600 dark:text-gray-300",
    barClassName: "bg-slate-100/90 dark:bg-gray-800/80",
  },
  setup: {
    id: "setup",
    Icon: Settings,
    iconClassName: "text-sky-600 dark:text-sky-400",
    barClassName: "bg-sky-100/70 dark:bg-sky-950/40",
  },
  runbook: {
    id: "runbook",
    Icon: BookOpen,
    iconClassName: "text-emerald-600 dark:text-emerald-400",
    barClassName: "bg-emerald-100/70 dark:bg-emerald-950/40",
  },
  run: {
    id: "run",
    Icon: Rocket,
    iconClassName: "text-blue-600 dark:text-blue-400",
    barClassName: "bg-blue-100/70 dark:bg-blue-950/40",
  },
  troubleshoot: {
    id: "troubleshoot",
    Icon: Wrench,
    iconClassName: "text-amber-600 dark:text-amber-400",
    barClassName: "bg-amber-100/70 dark:bg-amber-950/40",
  },
  agent: {
    id: "agent",
    Icon: Bot,
    iconClassName: "text-violet-600 dark:text-violet-400",
    barClassName: "bg-violet-100/80 dark:bg-violet-950/50",
  },
  integrations: {
    id: "integrations",
    Icon: Plug,
    iconClassName: "text-cyan-600 dark:text-cyan-400",
    barClassName: "bg-cyan-100/70 dark:bg-cyan-950/40",
  },
  group: {
    id: "group",
    Icon: Folder,
    iconClassName: "text-slate-500 dark:text-gray-400",
    barClassName: "bg-slate-100/70 dark:bg-gray-800/60",
  },
};

const DEFAULT_ROOT_PRESET = MARKDOWN_SECTION_PRESETS.runbook;
const DEFAULT_NESTED_PRESET = MARKDOWN_SECTION_PRESETS.group;

export function isMarkdownSectionPresetId(value: string): value is MarkdownSectionPresetId {
  return (MARKDOWN_SECTION_PRESET_IDS as readonly string[]).includes(value);
}

export function resolveMarkdownSectionPreset(presetId: string | undefined, depth: number): MarkdownSectionPreset {
  if (presetId && isMarkdownSectionPresetId(presetId)) {
    return MARKDOWN_SECTION_PRESETS[presetId];
  }

  return depth === 0 ? DEFAULT_ROOT_PRESET : DEFAULT_NESTED_PRESET;
}
