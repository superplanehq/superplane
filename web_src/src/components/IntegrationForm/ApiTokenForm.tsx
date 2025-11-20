import { Field, Label, ErrorMessage } from "../Fieldset/fieldset";
import { Input } from "../Input/input";
import { Link } from "../Link/link";
import { Text } from "../Text/text";
import type { BaseIntegrationFormProps } from "./types";

interface ApiTokenFormProps extends BaseIntegrationFormProps {
  organizationId: string;
  canvasId: string;
  isEditMode?: boolean;
}

export function ApiTokenForm({
  errors,
  setErrors,
  secretValue,
  setSecretValue,
  isEditMode = false,
}: ApiTokenFormProps) {
  return (
    <div className="space-y-4">
      <div className="text-sm font-medium text-gray-900 dark:text-white flex items-center justify-between">
        API Token
      </div>
      <Field>
        <Input
          type="password"
          placeholder={isEditMode ? "Enter new API token value" : "Enter your API token"}
          value={secretValue}
          className="w-full"
          onChange={(e) => {
            setSecretValue(e.target.value);
            if (errors.secretValue) {
              setErrors((prev) => ({ ...prev, secretValue: undefined }));
            }
          }}
        />
        {errors.secretValue && <ErrorMessage>{errors.secretValue}</ErrorMessage>}
      </Field>
    </div>
  );
}
