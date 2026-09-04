import { Text } from "@/components/Text/text";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { hostedProviderLabel } from "@/lib/hostedCredit";

import { parseDefaultModelKey } from "./hostedLLMDefaultModel";

export function HostedLLMDefaultModelField({
  options,
  value,
  saving,
  changed,
  onChange,
  onSave,
}: {
  options: Array<{ value: string; label: string }>;
  value: string;
  saving: boolean;
  changed: boolean;
  onChange: (value: string) => void;
  onSave: () => void;
}) {
  const parsedValue = parseDefaultModelKey(value);
  const valueOnAllowlist = options.some((option) => option.value === value);
  return (
    <div className="mt-8 max-w-2xl">
      <Label className="mb-2 block text-left" htmlFor="installation-llm-default-model">
        SuperPlane agent model
      </Label>
      <select
        id="installation-llm-default-model"
        data-testid="installation-llm-default-model"
        className="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm text-gray-900 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100"
        value={value}
        onChange={(event) => onChange(event.target.value)}
      >
        <option value="">No SuperPlane agent model</option>
        {value !== "" && !valueOnAllowlist ? (
          <option value={value}>
            {hostedProviderLabel(parsedValue.provider)} - {parsedValue.model}
          </option>
        ) : null}
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
      <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">
        {options.length === 0
          ? "Enable a provider and select models before you set a SuperPlane agent model."
          : "Run SuperPlane Agent uses this model."}
      </Text>
      <div className="mt-4">
        <Button
          type="button"
          data-testid="installation-llm-default-model-save"
          onClick={onSave}
          disabled={saving || !changed}
        >
          {saving ? "Saving..." : "Save SuperPlane agent model"}
        </Button>
      </div>
    </div>
  );
}
