import type { MermaidConfig } from "mermaid";
import type { ResolvedTheme } from "@/lib/themePreference";

const LIGHT_THEME_VARIABLES: NonNullable<MermaidConfig["themeVariables"]> = {
  primaryColor: "#ddd6fe",
  primaryTextColor: "#4c1d95",
  primaryBorderColor: "#7c3aed",
  secondaryColor: "#67e8f9",
  secondaryTextColor: "#164e63",
  secondaryBorderColor: "#0891b2",
  tertiaryColor: "#fcd34d",
  tertiaryTextColor: "#78350f",
  tertiaryBorderColor: "#d97706",
  lineColor: "#64748b",
  textColor: "#1e293b",
  nodeBorder: "#7c3aed",
  nodeTextColor: "#1e293b",
  clusterBkg: "#f8fafc",
  clusterBorder: "#cbd5e1",
  defaultLinkColor: "#7c3aed",
  fontSize: "13px",
  pie1: "#8b5cf6",
  pie2: "#06b6d4",
  pie3: "#f59e0b",
  pie4: "#10b981",
  pie5: "#f43f5e",
  pie6: "#3b82f6",
  pie7: "#eab308",
  pie8: "#14b8a6",
  pie9: "#ec4899",
  pie10: "#6366f1",
  pie11: "#84cc16",
  pie12: "#f97316",
  pieOpacity: "1",
  pieStrokeColor: "#ffffff",
  pieStrokeWidth: "1px",
  pieOuterStrokeColor: "#e2e8f0",
  pieTitleTextColor: "#1e293b",
  pieSectionTextColor: "#0f172a",
  pieLegendTextColor: "#334155",
};

const DARK_CARD_LABEL = "#e2e8f0";
const DARK_CARD_LINE = "#cbd5e1";

const DARK_THEME_VARIABLES: NonNullable<MermaidConfig["themeVariables"]> = {
  ...LIGHT_THEME_VARIABLES,
  lineColor: DARK_CARD_LINE,
  textColor: DARK_CARD_LABEL,
  pieTitleTextColor: DARK_CARD_LABEL,
  pieLegendTextColor: DARK_CARD_LABEL,
  signalColor: DARK_CARD_LINE,
  signalTextColor: DARK_CARD_LABEL,
  labelTextColor: DARK_CARD_LABEL,
  actorLineColor: DARK_CARD_LINE,
  loopTextColor: DARK_CARD_LABEL,
  sequenceNumberColor: DARK_CARD_LABEL,
};

export function mermaidInitializeConfig(resolvedTheme: ResolvedTheme): MermaidConfig {
  return {
    startOnLoad: false,
    theme: "base",
    securityLevel: "strict",
    fontFamily: "ui-sans-serif, system-ui, sans-serif",
    themeVariables: resolvedTheme === "dark" ? DARK_THEME_VARIABLES : LIGHT_THEME_VARIABLES,
  };
}
