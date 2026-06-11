import React from "react";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
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
}: {
  parameters: StartTemplateParameter[];
  parameterValues: Record<string, string | number | boolean>;
  onParameterValuesChange: React.Dispatch<React.SetStateAction<Record<string, string | number | boolean>>>;
}) {
  return (
    <div className="min-w-0 space-y-3">
      {parameters.map((param) => {
        if (!param.name || !param.type) return null;
        const id = `start-run-param-${param.name}`;
        const label = parameterDisplayLabel(param);
        return (
          <div key={param.name} className="min-w-0 space-y-1.5">
            {param.type === "boolean" ? (
              <div className="flex min-w-0 items-center gap-2">
                <Checkbox
                  id={id}
                  checked={Boolean(parameterValues[param.name])}
                  onCheckedChange={(checked) =>
                    onParameterValuesChange((prev) => ({
                      ...prev,
                      [param.name]: checked === true,
                    }))
                  }
                />
                <Label htmlFor={id} className="min-w-0 cursor-pointer">
                  {label}
                </Label>
              </div>
            ) : param.type === "select" ? (
              <>
                <Label htmlFor={id}>{label}</Label>
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
                  <SelectTrigger id={id} className="w-full min-w-0">
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
            ) : (
              <>
                <Label htmlFor={id}>{label}</Label>
                <Input
                  id={id}
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
