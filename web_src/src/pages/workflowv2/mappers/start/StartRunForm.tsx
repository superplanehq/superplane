import React from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { showErrorToast } from "@/lib/toast";
import { Checkbox } from "@/ui/checkbox";
import type { ParamDefinition } from "./paramSyntax";

export function StartRunForm({
  defs,
  onRun,
  onClose,
}: {
  defs: ParamDefinition[];
  onRun: (params: Record<string, unknown>) => Promise<void>;
  onClose: () => void;
}) {
  const { values, updateValue } = useStartRunParamValues(defs);
  const { isSubmitting, handleSubmit } = useStartRunSubmit({ defs, values, onRun, onClose });

  return (
    <form className="space-y-4" onSubmit={handleSubmit}>
      <div className="space-y-4">
        {defs.map((def) => {
          const fieldId = `start-run-${def.path.replace(/[^a-zA-Z0-9_-]+/g, "-")}`;
          const label = labelFor(def);

          if (def.type === "boolean") {
            return (
              <div key={def.path} className="flex items-center gap-2">
                <Checkbox
                  id={fieldId}
                  checked={Boolean(values[def.path])}
                  onCheckedChange={(checked) => updateValue(def.path, checked === true)}
                  disabled={isSubmitting}
                />
                <Label htmlFor={fieldId}>{label}</Label>
              </div>
            );
          }

          if (def.type === "select") {
            return (
              <div key={def.path} className="space-y-2">
                <Label htmlFor={fieldId}>{label}</Label>
                <Select
                  value={String(values[def.path] ?? "")}
                  onValueChange={(next) => updateValue(def.path, next)}
                  disabled={isSubmitting}
                >
                  <SelectTrigger id={fieldId} className="w-full">
                    <SelectValue placeholder={`Select ${label.toLowerCase()}`} />
                  </SelectTrigger>
                  <SelectContent>
                    {def.values.map((option) => (
                      <SelectItem key={option} value={option}>
                        {option}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            );
          }

          return (
            <div key={def.path} className="space-y-2">
              <Label htmlFor={fieldId}>{label}</Label>
              <Input
                id={fieldId}
                type={def.type === "number" ? "number" : "text"}
                value={values[def.path] === undefined || values[def.path] === null ? "" : String(values[def.path])}
                onChange={(event) => {
                  const next = event.target.value;
                  updateValue(def.path, def.type === "number" ? next : next);
                }}
                disabled={isSubmitting}
              />
            </div>
          );
        })}
      </div>
      <div className="flex items-center justify-end gap-2">
        <Button type="button" variant="outline" onClick={onClose} disabled={isSubmitting}>
          Cancel
        </Button>
        <LoadingButton
          type="submit"
          data-testid="start-run-submit-button"
          loading={isSubmitting}
          loadingText="Running..."
        >
          Run
        </LoadingButton>
      </div>
    </form>
  );
}

type RunParamValues = Record<string, string | number | boolean>;

function useStartRunParamValues(defs: ParamDefinition[]) {
  const [values, setValues] = React.useState<RunParamValues>(() => initialValues(defs));

  const updateValue = (path: string, next: string | number | boolean) => {
    setValues((current) => ({ ...current, [path]: next }));
  };

  return { values, updateValue };
}

function useStartRunSubmit({
  defs,
  values,
  onRun,
  onClose,
}: {
  defs: ParamDefinition[];
  values: RunParamValues;
  onRun: (params: Record<string, unknown>) => Promise<void>;
  onClose: () => void;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState(false);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    const validationError = validateValues(defs, values);
    if (validationError) {
      showErrorToast(validationError);
      return;
    }

    setIsSubmitting(true);
    try {
      await onRun(buildRunParams(defs, values));
      onClose();
    } catch {
      // Keep the modal open so users can retry.
    } finally {
      setIsSubmitting(false);
    }
  };

  return { isSubmitting, handleSubmit };
}

function initialValues(defs: ParamDefinition[]): RunParamValues {
  const values: RunParamValues = {};
  for (const def of defs) {
    if (def.default !== undefined && def.default !== null) {
      values[def.path] = def.default as string | number | boolean;
      continue;
    }
    switch (def.type) {
      case "string":
        values[def.path] = "";
        break;
      case "number":
        values[def.path] = "";
        break;
      case "boolean":
        values[def.path] = false;
        break;
      case "select":
        values[def.path] = def.values[0] ?? "";
        break;
    }
  }
  return values;
}

function validateValues(defs: ParamDefinition[], values: RunParamValues): string | null {
  for (const def of defs) {
    const raw = values[def.path];
    const hasDefault = def.default !== undefined && def.default !== null;

    if (def.type === "string" || def.type === "select") {
      const text = typeof raw === "string" ? raw.trim() : "";
      if (def.required && !hasDefault && text === "") {
        return `${labelFor(def)} is required`;
      }
      continue;
    }

    if (def.type === "number") {
      if (raw === "" || raw === undefined || raw === null) {
        if (def.required && !hasDefault) {
          return `${labelFor(def)} is required`;
        }
        continue;
      }
      const numberValue = typeof raw === "number" ? raw : Number(raw);
      if (!Number.isFinite(numberValue)) {
        return `${labelFor(def)} must be a number`;
      }
      continue;
    }
  }
  return null;
}

function labelFor(def: ParamDefinition): string {
  return def.title.trim() !== "" ? def.title : def.path;
}

function buildRunParams(defs: ParamDefinition[], values: RunParamValues): Record<string, unknown> {
  const params: Record<string, unknown> = {};
  for (const def of defs) {
    const raw = values[def.path];
    switch (def.type) {
      case "string":
      case "select":
        params[def.path] = typeof raw === "string" ? raw : String(raw ?? "");
        break;
      case "number": {
        if (raw === "" || raw === undefined || raw === null) {
          if (def.default !== undefined && def.default !== null) {
            params[def.path] = def.default;
          }
          break;
        }
        params[def.path] = typeof raw === "number" ? raw : Number(raw);
        break;
      }
      case "boolean":
        params[def.path] = Boolean(raw);
        break;
    }
  }
  return params;
}
