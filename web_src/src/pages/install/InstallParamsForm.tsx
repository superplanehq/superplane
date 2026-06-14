import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { InstallParam } from "./types";

interface InstallParamsFormProps {
  params: InstallParam[];
  values: Record<string, string>;
  onChange: (name: string, value: string) => void;
}

export function InstallParamsForm({ params, values, onChange }: InstallParamsFormProps) {
  if (params.length === 0) {
    return null;
  }

  return (
    <div className="space-y-4">
      <div className="border-t pt-4">
        <h3 className="text-sm font-medium text-slate-700 mb-3">Configuration</h3>
        <div className="space-y-4">
          {params.map((param) => (
            <div key={param.name} className="space-y-1.5">
              <Label htmlFor={`install-param-${param.name}`}>
                {param.label}
                {param.required && <span className="text-red-500 ml-0.5">*</span>}
              </Label>
              <Input
                id={`install-param-${param.name}`}
                data-testid={`install-param-${param.name}`}
                value={values[param.name] ?? param.default ?? ""}
                placeholder={param.placeholder}
                onChange={(e) => onChange(param.name, e.target.value)}
              />
              {param.description && <p className="text-xs text-slate-500">{param.description}</p>}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
