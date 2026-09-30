import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ChevronDown, ChevronRight } from "lucide-react";
import { useState } from "react";

import {
  BADGE_COLOR_LABELS,
  BADGE_COLOR_SLOTS,
  BADGE_THEMES,
  type BadgeColorSlot,
  type BadgeColors,
  type BadgeTheme,
  badgeThemeColors,
  badgeThemeLabel,
  isBadgeColor,
} from "../../lib/badgeThemes";

const COPY = {
  themeLabel: "Theme",
  customized: "Custom",
  reset: "Reset colors",
  colorsLabel: "Colors",
  colorsHelper: "Change any color to build your own theme.",
} as const;

/**
 * Theme presets plus the color fields they fill. Picking a preset replaces
 * every field, and editing one field keeps the rest of the preset.
 */
export function PublicBadgeThemeEditor({
  theme,
  colors,
  customized,
  locked,
  onThemeChange,
  onColorChange,
  onReset,
}: {
  theme: BadgeTheme;
  colors: BadgeColors;
  customized: boolean;
  locked: boolean;
  onThemeChange: (theme: BadgeTheme) => void;
  onColorChange: (slot: BadgeColorSlot, color: string) => void;
  onReset: () => void;
}) {
  const [open, setOpen] = useState(false);
  const showColors = open || customized;

  return (
    <div className="space-y-3">
      <div className="flex min-h-5 items-center justify-between gap-3">
        <div className="flex items-baseline gap-2">
          <Label>{COPY.themeLabel}</Label>
          <span className="text-[12px] text-muted-foreground" data-testid="factory-settings-public-badge-theme-name">
            {customized ? COPY.customized : badgeThemeLabel(theme)}
          </span>
        </div>
        {customized ? (
          // A text action, not a button: a sized button makes the row taller
          // and moves the rest of the section down when a color changes.
          <button
            type="button"
            className="text-[12px] text-muted-foreground underline hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
            disabled={locked}
            onClick={onReset}
            data-testid="factory-settings-public-badge-reset"
          >
            {COPY.reset}
          </button>
        ) : null}
      </div>

      <div className="flex flex-wrap gap-2" data-testid="factory-settings-public-badge-themes">
        {BADGE_THEMES.map((option) => (
          <ThemeSwatch
            key={option.value}
            label={option.label}
            colors={option.colors}
            selected={!customized && option.value === theme}
            disabled={locked}
            onSelect={() => onThemeChange(option.value)}
          />
        ))}
      </div>

      <div>
        <button
          type="button"
          className="flex items-center gap-1 text-[12px] text-muted-foreground hover:text-foreground"
          onClick={() => setOpen(!showColors)}
          aria-expanded={showColors}
          data-testid="factory-settings-public-badge-colors-toggle"
        >
          {showColors ? <ChevronDown className="size-3.5" /> : <ChevronRight className="size-3.5" />}
          {COPY.colorsLabel}
        </button>
        {showColors ? (
          <div className="mt-3 space-y-3">
            <div className="grid gap-3 sm:grid-cols-2">
              {BADGE_COLOR_SLOTS.map((slot) => (
                <ColorField
                  key={slot}
                  slot={slot}
                  value={colors[slot]}
                  fallback={badgeThemeColors(theme)[slot]}
                  locked={locked}
                  onChange={(color) => onColorChange(slot, color)}
                />
              ))}
            </div>
            <p className="text-[12px] text-muted-foreground">{COPY.colorsHelper}</p>
          </div>
        ) : null}
      </div>
    </div>
  );
}

function ThemeSwatch({
  label,
  colors,
  selected,
  disabled,
  onSelect,
}: {
  label: string;
  colors: BadgeColors;
  selected: boolean;
  disabled: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      aria-pressed={selected}
      title={label}
      disabled={disabled}
      onClick={onSelect}
      style={{ backgroundColor: colors.bg, borderColor: selected ? colors.accent : colors.border }}
      className={`flex h-9 w-12 flex-col items-center justify-center gap-1 rounded-md border transition disabled:cursor-not-allowed disabled:opacity-50 ${
        selected ? "ring-2 ring-ring ring-offset-2 ring-offset-background" : "hover:opacity-80"
      }`}
    >
      <span className="block h-1.5 w-6 rounded-full" style={{ backgroundColor: colors.accent }} />
      <span className="block h-1 w-4 rounded-full" style={{ backgroundColor: colors.muted }} />
    </button>
  );
}

function ColorField({
  slot,
  value,
  fallback,
  locked,
  onChange,
}: {
  slot: BadgeColorSlot;
  value: string;
  fallback: string;
  locked: boolean;
  onChange: (color: string) => void;
}) {
  const label = BADGE_COLOR_LABELS[slot];
  const shown = isBadgeColor(value) ? value.toLowerCase() : fallback;
  return (
    <div className="flex items-center gap-2">
      <Input
        type="color"
        aria-label={`${label} swatch`}
        value={shown}
        disabled={locked}
        onChange={(event) => onChange(event.target.value)}
        className="h-8 w-8 shrink-0 cursor-pointer p-0.5"
        data-testid={`factory-settings-public-badge-color-swatch-${slot}`}
      />
      <div className="min-w-0 flex-1">
        <Label htmlFor={`factory-settings-public-badge-color-${slot}`} className="text-[11px] text-muted-foreground">
          {label}
        </Label>
        <Input
          id={`factory-settings-public-badge-color-${slot}`}
          data-testid={`factory-settings-public-badge-color-${slot}`}
          value={value}
          maxLength={7}
          disabled={locked}
          onChange={(event) => onChange(event.target.value.trim())}
          className="h-7 font-mono text-[12px]"
          autoComplete="off"
          spellCheck={false}
        />
      </div>
    </div>
  );
}
