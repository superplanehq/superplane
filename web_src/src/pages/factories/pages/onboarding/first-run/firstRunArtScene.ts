export type FirstRunArtMode = "school" | "globe";

export type FirstRunArtPill = {
  label: string;
  value: string;
  light: boolean;
};

export type FirstRunArtInk = "light" | "dark";

/**
 * Text drawn over the art follows the arrow color: dark arrows mean a light
 * background, so the text goes dark too.
 */
export function artInk(scene: Pick<FirstRunArtScene, "arrowColor"> | undefined): FirstRunArtInk {
  if (!scene) return "light";
  const hex = scene.arrowColor.replace("#", "");
  if (hex.length !== 6) return "light";
  const [r, g, b] = [0, 2, 4].map((offset) => parseInt(hex.slice(offset, offset + 2), 16));
  const brightness = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
  return brightness < 0.5 ? "dark" : "light";
}

export type FirstRunArtScene = {
  mode: FirstRunArtMode;
  background: string;
  arrowColor: string;
  /** Globe particle count. School leaves this unset. */
  count?: number;
  /** Optional per-scene arrow size; the renderer default applies when omitted. */
  arrowScale?: number;
  pill?: FirstRunArtPill;
  badges?: { discover: string; verify: string };
};
