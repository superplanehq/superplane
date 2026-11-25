import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ChevronRight } from "lucide-react";

import {
  webhookIntakeExampleBody,
  webhookIntakeFieldPathErrors,
  type WebhookIntakeFieldPaths,
} from "./webhookIntakeFieldPaths";
import { WEBHOOK_INTAKE_FIELDS_COPY } from "./webhookIntakeSetupCopy";

const FIELDS = [
  { key: "title", label: WEBHOOK_INTAKE_FIELDS_COPY.titleLabel, helper: WEBHOOK_INTAKE_FIELDS_COPY.titleHelper },
  {
    key: "description",
    label: WEBHOOK_INTAKE_FIELDS_COPY.descriptionLabel,
    helper: WEBHOOK_INTAKE_FIELDS_COPY.descriptionHelper,
  },
  { key: "link", label: WEBHOOK_INTAKE_FIELDS_COPY.linkLabel, helper: WEBHOOK_INTAKE_FIELDS_COPY.linkHelper },
] as const;

const PLACEHOLDERS: WebhookIntakeFieldPaths = { title: "title", description: "description", link: "url" };

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
    <div className="space-y-3" data-testid={`${idPrefix}-field-paths`}>
      {FIELDS.map((field) => {
        const id = `${idPrefix}-${field.key}-path`;
        const error = errors[field.key];
        return (
          <div key={field.key} className="grid gap-x-3 gap-y-1 sm:grid-cols-[7rem_minmax(0,1fr)] sm:items-center">
            <Label htmlFor={id}>{field.label}</Label>
            <Input
              id={id}
              value={paths[field.key]}
              placeholder={PLACEHOLDERS[field.key]}
              disabled={disabled}
              spellCheck={false}
              autoComplete="off"
              className="font-mono text-[13px]"
              aria-invalid={Boolean(error)}
              aria-describedby={`${id}-helper`}
              onChange={(event) => onChange({ ...paths, [field.key]: event.target.value })}
              data-testid={id}
            />
            <p
              id={`${id}-helper`}
              className={
                error ? "text-xs text-destructive sm:col-start-2" : "text-xs text-muted-foreground sm:col-start-2"
              }
            >
              {error ?? field.helper}
            </p>
          </div>
        );
      })}
      <details className="group rounded-md border border-border">
        <summary className="flex cursor-pointer list-none items-center gap-1.5 px-3 py-2 text-xs font-medium text-muted-foreground hover:text-foreground [&::-webkit-details-marker]:hidden">
          <ChevronRight className="size-3.5 transition-transform group-open:rotate-90" aria-hidden />
          {WEBHOOK_INTAKE_FIELDS_COPY.exampleLabel}
        </summary>
        <pre
          className="overflow-x-auto border-t border-border bg-muted/40 px-3 py-2 text-[12px] leading-relaxed text-foreground"
          data-testid={`${idPrefix}-example-body`}
        >
          {webhookIntakeExampleBody(paths)}
        </pre>
      </details>
    </div>
  );
}
