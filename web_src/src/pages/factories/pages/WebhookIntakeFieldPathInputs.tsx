import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

import {
  webhookIntakeExampleBody,
  webhookIntakeFieldPathErrors,
  type WebhookIntakeFieldPaths,
} from "./webhookIntakeFieldPaths";
import { WEBHOOK_INTAKE_FIELDS_COPY } from "./webhookIntakeSetupCopy";

const FIELDS = [
  {
    key: "title",
    label: WEBHOOK_INTAKE_FIELDS_COPY.titleLabel,
    helper: WEBHOOK_INTAKE_FIELDS_COPY.titleHelper,
    placeholder: "title",
  },
  {
    key: "description",
    label: WEBHOOK_INTAKE_FIELDS_COPY.descriptionLabel,
    helper: WEBHOOK_INTAKE_FIELDS_COPY.descriptionHelper,
    placeholder: "description",
  },
  {
    key: "link",
    label: WEBHOOK_INTAKE_FIELDS_COPY.linkLabel,
    helper: WEBHOOK_INTAKE_FIELDS_COPY.linkHelper,
    placeholder: "url",
  },
] as const;

export function WebhookIntakeFieldPathInputs({
  idPrefix,
  paths,
  disabled,
  onChange,
}: {
  idPrefix: string;
  paths: WebhookIntakeFieldPaths;
  disabled?: boolean;
  onChange: (paths: WebhookIntakeFieldPaths) => void;
}) {
  const errors = webhookIntakeFieldPathErrors(paths);
  return (
    <div className="space-y-4" data-testid={`${idPrefix}-field-paths`}>
      {FIELDS.map((field) => {
        const id = `${idPrefix}-${field.key}-path`;
        const error = errors[field.key];
        return (
          <div key={field.key} className="space-y-1.5">
            <Label htmlFor={id}>{field.label}</Label>
            <Input
              id={id}
              value={paths[field.key]}
              placeholder={field.placeholder}
              disabled={disabled}
              spellCheck={false}
              aria-invalid={Boolean(error)}
              aria-describedby={`${id}-helper`}
              onChange={(event) => onChange({ ...paths, [field.key]: event.target.value })}
              data-testid={id}
            />
            <p id={`${id}-helper`} className={error ? "text-xs text-destructive" : "text-xs text-muted-foreground"}>
              {error ?? field.helper}
            </p>
          </div>
        );
      })}
      <div>
        <p className="mb-1.5 text-[11px] font-medium uppercase tracking-[0.04em] text-muted-foreground">
          {WEBHOOK_INTAKE_FIELDS_COPY.exampleLabel}
        </p>
        <pre
          className="overflow-x-auto rounded-md border border-border bg-muted/40 px-3 py-2 text-[12px] leading-relaxed text-foreground"
          data-testid={`${idPrefix}-example-body`}
        >
          {webhookIntakeExampleBody(paths)}
        </pre>
      </div>
    </div>
  );
}
