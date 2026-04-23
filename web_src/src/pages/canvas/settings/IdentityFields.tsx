import { Field, Fieldset, Label } from "@/components/Fieldset/fieldset";
import { Input } from "@/components/Input/input";
import { Textarea } from "@/components/ui/textarea";

type IdentityFieldsProps = {
  name: string;
  description: string;
  onNameChange: (value: string) => void;
  onDescriptionChange: (value: string) => void;
  canUpdateCanvas: boolean;
};

export function IdentityFields({
  name,
  description,
  onNameChange,
  onDescriptionChange,
  canUpdateCanvas,
}: IdentityFieldsProps) {
  return (
    <Fieldset className="space-y-5">
      <Field className="space-y-3">
        <Label htmlFor="canvas-settings-name-input" className="block text-sm font-medium text-slate-900">
          Canvas Name
        </Label>
        <Input
          id="canvas-settings-name-input"
          type="text"
          value={name}
          onChange={(event) => onNameChange(event.target.value)}
          disabled={!canUpdateCanvas}
        />
      </Field>

      <Field className="space-y-3">
        <Label htmlFor="canvas-settings-description-input" className="block text-sm font-medium text-slate-900">
          Description
        </Label>
        <Textarea
          id="canvas-settings-description-input"
          value={description}
          onChange={(event) => onDescriptionChange(event.target.value)}
          disabled={!canUpdateCanvas}
          rows={4}
          placeholder="Describe canvas…"
        />
      </Field>
    </Fieldset>
  );
}
