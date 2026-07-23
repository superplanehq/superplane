import React from "react";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { Checkbox } from "@/ui/checkbox";

import {
  parameterDisplayLabel,
  parameterInputPlaceholder,
  selectOptionValues,
  type StartTemplateParameter,
} from "./templatePayload";

export function StartRunParameterFields({
  parameters,
  parameterValues,
  onParameterValuesChange,
  showLabels = true,
  fillAvailableHeight = false,
}: {
  parameters: StartTemplateParameter[];
  parameterValues: Record<string, string | number | boolean>;
  onParameterValuesChange: React.Dispatch<React.SetStateAction<Record<string, string | number | boolean>>>;
  /** Visually show labels. Hidden labels remain associated for accessibility. */
  showLabels?: boolean;
  /**
   * Stretch `text` parameters to fill leftover vertical space (inline console
   * forms). Modal / compact embeds leave this false so rows stay content-sized.
   */
  fillAvailableHeight?: boolean;
}) {
  const idPrefix = React.useId();
  return (
    <div
      className={cn("min-w-0", fillAvailableHeight ? "flex h-full min-h-0 flex-col gap-3" : "space-y-3")}
      data-fill-available-height={fillAvailableHeight || undefined}
    >
      {parameters.map((param) => {
        if (!param.name || !param.type) return null;
        const id = `${idPrefix}-start-run-param-${param.name}`;
        // Stable hook for E2E tests; `id` carries a `useId` prefix so that
        // multiple forms on one page never share duplicate element ids.
        const testId = `start-run-param-${param.name}`;
        const label = parameterDisplayLabel(param);
        const stretchText = fillAvailableHeight && param.type === "text";
        return (
          <div key={param.name} className={cn("min-w-0 space-y-1.5", stretchText && "flex min-h-0 flex-1 flex-col")}>
            {param.type === "boolean" ? (
              <div className="flex min-w-0 items-center gap-2">
                <Checkbox
                  id={id}
                  data-testid={testId}
                  checked={Boolean(parameterValues[param.name])}
                  onCheckedChange={(checked) =>
                    onParameterValuesChange((prev) => ({
                      ...prev,
                      [param.name]: checked === true,
                    }))
                  }
                />
                <Label htmlFor={id} className={parameterLabelClassName(showLabels, "min-w-0 cursor-pointer")}>
                  {label}
                </Label>
              </div>
            ) : param.type === "select" ? (
              <>
                <Label htmlFor={id} className={parameterLabelClassName(showLabels)}>
                  {label}
                </Label>
                <Select
                  value={String(parameterValues[param.name] ?? "")}
                  onValueChange={(val) =>
                    onParameterValuesChange((prev) => ({
                      ...prev,
                      [param.name]: val,
                    }))
                  }
                  disabled={selectOptionValues(param).length === 0}
                >
                  <SelectTrigger id={id} data-testid={testId} className="w-full min-w-0">
                    <SelectValue
                      placeholder={selectOptionValues(param).length === 0 ? "No options configured" : `Select ${label}`}
                    />
                  </SelectTrigger>
                  <SelectContent className="max-h-60">
                    {(param.options ?? []).map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label || opt.value}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </>
            ) : param.type === "text" ? (
              <>
                <Label
                  htmlFor={id}
                  className={parameterLabelClassName(showLabels, stretchText ? "shrink-0" : undefined)}
                >
                  {label}
                </Label>
                <Textarea
                  id={id}
                  data-testid={testId}
                  placeholder={parameterInputPlaceholder(param, label)}
                  value={String(parameterValues[param.name] ?? "")}
                  rows={stretchText ? undefined : 5}
                  className={stretchText ? "min-h-0 flex-1 resize-none [field-sizing:fixed]" : undefined}
                  onChange={(e) =>
                    onParameterValuesChange((prev) => ({
                      ...prev,
                      [param.name]: e.target.value,
                    }))
                  }
                />
              </>
            ) : (
              <>
                <Label htmlFor={id} className={parameterLabelClassName(showLabels)}>
                  {label}
                </Label>
                <Input
                  id={id}
                  data-testid={testId}
                  type={param.type === "number" ? "number" : "text"}
                  placeholder={parameterInputPlaceholder(param, label)}
                  value={String(parameterValues[param.name] ?? "")}
                  onChange={(e) =>
                    onParameterValuesChange((prev) => ({
                      ...prev,
                      [param.name]: e.target.value,
                    }))
                  }
                />
              </>
            )}
          </div>
        );
      })}
    </div>
  );
}

function parameterLabelClassName(showLabels: boolean, visibleClassName?: string): string | undefined {
  return showLabels ? visibleClassName : "sr-only";
}
