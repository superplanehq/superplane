import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import { useFactoryLineRunnerModels } from "@/hooks/useFactoryLineRunnerModels";

import { DRAFT_START_MODEL_AUTO } from "./draftStartModel";

export function DraftStartModelSelect({
  organizationId,
  factoryId,
  lineName,
  value,
  onChange,
  disabled = false,
}: {
  organizationId?: string;
  factoryId?: string;
  lineName?: string;
  value: string;
  onChange: (next: string) => void;
  disabled?: boolean;
}) {
  const models = useFactoryLineRunnerModels(organizationId, factoryId, lineName);

  return (
    <Select value={value} onValueChange={onChange} disabled={disabled}>
      <SelectTrigger size="sm" className="w-[11.5rem]" aria-label="Model" data-testid="split-run-draft-model">
        <SelectValue placeholder="Auto" />
      </SelectTrigger>
      <SelectContent position="popper" className="max-h-60">
        <SelectItem value={DRAFT_START_MODEL_AUTO}>Auto</SelectItem>
        {(models.data ?? []).map((model) => {
          const id = model.id ?? "";
          if (id === "") {
            return null;
          }
          return (
            <SelectItem key={id} value={id}>
              {model.name || id}
            </SelectItem>
          );
        })}
      </SelectContent>
    </Select>
  );
}
