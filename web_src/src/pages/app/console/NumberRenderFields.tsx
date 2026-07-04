import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import { NUMBER_PANEL_FORMATS } from "./numberPanelFormConstants";
import type { WidgetColumnFormat, WidgetNumberRender } from "./widget/types";

export type NumberRenderFieldVariant = "default" | "compact";

type VariantStyles = {
  label: string;
  field: string;
  gridGap: string;
  optionalSuffix: string;
};

// The Number panel form ("default") and the per-metric editor ("compact") use
// the same fields with slightly different typography and spacing, so the
// visual differences live here instead of being duplicated per call site.
const VARIANT_STYLES: Record<NumberRenderFieldVariant, VariantStyles> = {
  default: {
    label: "text-xs font-medium text-slate-600",
    field: "space-y-1.5",
    gridGap: "gap-3",
    optionalSuffix: " (optional)",
  },
  compact: {
    label: "text-[10px] font-medium uppercase tracking-wide text-slate-500",
    field: "space-y-1",
    gridGap: "gap-2",
    optionalSuffix: "",
  },
};

type NumberRenderFieldProps = {
  render: WidgetNumberRender;
  onChange: (patch: Partial<WidgetNumberRender>) => void;
  variant?: NumberRenderFieldVariant;
};

export function NumberFormatField({ render, onChange, variant = "default" }: NumberRenderFieldProps) {
  const styles = VARIANT_STYLES[variant];
  return (
    <div className={styles.field}>
      <Label className={styles.label}>Format</Label>
      <Select
        value={render.format ?? "__none__"}
        onValueChange={(v) => onChange({ format: v === "__none__" ? undefined : (v as WidgetColumnFormat) })}
      >
        <SelectTrigger className="w-full">
          <SelectValue placeholder="Default" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="__none__">Default</SelectItem>
          {NUMBER_PANEL_FORMATS.map((f) => (
            <SelectItem key={f} value={f}>
              {f}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

export function NumberLabelField({ render, onChange, variant = "default" }: NumberRenderFieldProps) {
  const styles = VARIANT_STYLES[variant];
  return (
    <div className={styles.field}>
      <Label className={styles.label}>Label{styles.optionalSuffix}</Label>
      <Input
        value={render.label ?? ""}
        onChange={(e) => onChange({ label: e.target.value || undefined })}
        placeholder="e.g. Total duration"
      />
    </div>
  );
}

export function NumberPrefixSuffixFields({ render, onChange, variant = "default" }: NumberRenderFieldProps) {
  const styles = VARIANT_STYLES[variant];
  return (
    <div className={`grid grid-cols-2 ${styles.gridGap}`}>
      <div className={styles.field}>
        <Label className={styles.label}>Prefix{styles.optionalSuffix}</Label>
        <Input
          value={render.prefix ?? ""}
          onChange={(e) => onChange({ prefix: e.target.value || undefined })}
          placeholder="e.g. R$"
        />
      </div>
      <div className={styles.field}>
        <Label className={styles.label}>Suffix{styles.optionalSuffix}</Label>
        <Input
          value={render.suffix ?? ""}
          onChange={(e) => onChange({ suffix: e.target.value || undefined })}
          placeholder="e.g. MWh"
        />
      </div>
    </div>
  );
}

export function NumberSparklineField({ render, onChange, variant = "default" }: NumberRenderFieldProps) {
  const styles = VARIANT_STYLES[variant];
  return (
    <div className={styles.field}>
      <Label className={styles.label}>Sparkline field{styles.optionalSuffix}</Label>
      <Input
        value={render.sparklineField ?? ""}
        onChange={(e) => onChange({ sparklineField: e.target.value || undefined })}
        placeholder="e.g. createdAt"
      />
    </div>
  );
}
