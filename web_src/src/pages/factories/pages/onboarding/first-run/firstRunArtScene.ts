export type FirstRunArtMode = "school" | "globe";

export type FirstRunArtPill = {
  label: string;
  value: string;
  light: boolean;
};

export type FirstRunArtScene = {
  mode: FirstRunArtMode;
  background: string;
  arrowColor: string;
  /** Globe particle count. School leaves this unset. */
  count?: number;
  pill?: FirstRunArtPill;
  badges?: { discover: string; verify: string };
};
