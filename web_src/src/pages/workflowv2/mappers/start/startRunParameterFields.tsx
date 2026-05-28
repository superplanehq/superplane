import React from "react";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/ui/checkbox";

import { parameterDisplayLabel, type StartTemplateParameter } from "./templatePayload";

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
    <div className="space-y-3">
      {parameters.map((param) => {
        if (!param.name || !param.type) return null;
        const id = `start-run-param-${param.name}`;
        const label = parameterDisplayLabel(param);
        return (
          <div key={param.name} className="space-y-1.5">
            {param.type === "boolean" ? (
              <div className="flex items-center gap-2">
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
                <Label htmlFor={id} className="cursor-pointer">
                  {label}
                </Label>
              </div>
            ) : (
              <>
                <Label htmlFor={id}>{label}</Label>
                <Input
                  id={id}
                  type={param.type === "number" ? "number" : "text"}
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
