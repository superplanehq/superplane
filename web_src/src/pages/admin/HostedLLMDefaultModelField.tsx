import { Text } from "@/components/Text/text";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { hostedProviderLabel } from "@/lib/hostedCredit";

import { parseDefaultModelKey } from "./hostedLLMDefaultModel";

const NONE_VALUE = "none";

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
  const selectValue = value === "" ? NONE_VALUE : value;

  return (
    <div className="mt-8 max-w-2xl">
      <Label className="mb-2 block text-left" htmlFor="installation-llm-default-model">
        SuperPlane agent model
      </Label>
      <Select value={selectValue} onValueChange={(next) => onChange(next === NONE_VALUE ? "" : next)}>
        <SelectTrigger
          id="installation-llm-default-model"
          data-testid="installation-llm-default-model"
          className="h-10 w-full"
        >
          <SelectValue placeholder="No SuperPlane agent model" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={NONE_VALUE}>No SuperPlane agent model</SelectItem>
          {value !== "" && !valueOnAllowlist ? (
            <SelectItem value={value}>
              {hostedProviderLabel(parsedValue.provider)} - {parsedValue.model}
            </SelectItem>
          ) : null}
          {options.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">
        {options.length === 0
          ? "Save a provider API key and select models before you set a SuperPlane agent model."
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
