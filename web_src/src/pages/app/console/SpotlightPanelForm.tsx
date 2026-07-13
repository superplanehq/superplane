import type { ReactNode } from "react";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

import { useConsoleContext } from "./ConsoleContext";
import { DataSourceForm } from "./DataSourceForm";
import { applySourceDefaults, checksAreStages, type SpotlightPanelContent } from "./spotlightContent";
import { staticFieldsForDataSource } from "./widget/staticFieldCatalogs";
import { useMemoryCatalog, type MemoryFieldSummary } from "./widget/useMemoryCatalog";
import type { WidgetDataSource } from "./widget/types";

const FIELD_LIST_ID = "spotlight-field-options";

export function SpotlightPanelForm({
  value,
  onChange,
}: {
  value: SpotlightPanelContent;
  onChange: (next: SpotlightPanelContent) => void;
}) {
  const ctx = useConsoleContext();
  const canvasId = ctx?.canvasId;
  const namespace = value.dataSource.kind === "memory" ? value.dataSource.namespace : "";
  const { fields: memoryFields } = useMemoryCatalog(canvasId, namespace);
  const fieldOptions = resolveFieldCatalog(value, memoryFields).map((f) => f.field);

  // Swap slot defaults when the source kind changes; leave them untouched when
  // only kind-specific fields (namespace, node, limit) are edited.
  const handleDataSource = (next: WidgetDataSource) => {
    if (next.kind !== value.dataSource.kind) {
      onChange(applySourceDefaults(value, next));
      return;
    }
    onChange({ ...value, dataSource: next });
  };

  return (
    <div className="flex flex-col gap-6">
      <Section title="Data source" hint="Which record to spotlight — the banner shows the top row.">
        <DataSourceForm value={value.dataSource} onChange={(next) => handleDataSource(next as WidgetDataSource)} />
      </Section>

      <HeaderSection value={value} onChange={onChange} fieldOptions={fieldOptions} />
      <HeadlineSection value={value} onChange={onChange} fieldOptions={fieldOptions} />
      <WhoSection value={value} onChange={onChange} fieldOptions={fieldOptions} />
      <TimingSection value={value} onChange={onChange} fieldOptions={fieldOptions} />
      <SecondaryPersonSection value={value} onChange={onChange} fieldOptions={fieldOptions} />
      <ChecksSection value={value} onChange={onChange} fieldOptions={fieldOptions} />

      {fieldOptions.length > 0 ? (
        <datalist id={FIELD_LIST_ID}>
          {fieldOptions.map((f) => (
            <option key={f} value={f} />
          ))}
        </datalist>
      ) : null}
    </div>
  );
}

/**
 * Field suggestions for the current source: discovered memory keys, or the
 * static run/execution catalogs the table form uses. Empty for memory until a
 * namespace with entries is chosen, in which case the inputs stay free-text.
 */
function resolveFieldCatalog(value: SpotlightPanelContent, memoryFields: MemoryFieldSummary[]): MemoryFieldSummary[] {
  if (value.dataSource.kind === "memory") return memoryFields;
  return staticFieldsForDataSource(value.dataSource.kind);
}

function Section({ title, hint, children }: { title: string; hint?: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-3">
      <div className="flex flex-col gap-0.5">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-gray-400">{title}</h3>
        {hint ? <p className="text-[11px] text-slate-400 dark:text-gray-500">{hint}</p> : null}
      </div>
      {children}
    </section>
  );
}

function Field({ label, hint, children }: { label: string; hint?: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label className="text-xs font-medium text-slate-600 dark:text-gray-300">{label}</Label>
      {children}
      {hint ? <p className="text-[11px] text-slate-400 dark:text-gray-500">{hint}</p> : null}
    </div>
  );
}

/** A single field-mapping input bound to one string key of the content, with field autocomplete. */
function SlotField({
  label,
  hint,
  placeholder,
  fieldKey,
  value,
  onChange,
  fieldOptions,
  invalid,
}: {
  label: string;
  hint?: string;
  placeholder?: string;
  fieldKey: keyof SpotlightPanelContent;
  value: SpotlightPanelContent;
  onChange: (next: SpotlightPanelContent) => void;
  fieldOptions: string[];
  invalid?: boolean;
}) {
  return (
    <Field label={label} hint={hint}>
      <Input
        value={(value[fieldKey] as string | undefined) ?? ""}
        onChange={(e) => onChange({ ...value, [fieldKey]: e.target.value })}
        placeholder={placeholder}
        aria-invalid={invalid}
        list={fieldOptions.length > 0 ? FIELD_LIST_ID : undefined}
      />
    </Field>
  );
}

function HeaderSection({ value, onChange, fieldOptions }: SectionProps) {
  return (
    <Section title="Header" hint="The eyebrow and status pill at the top of the banner.">
      <Field label="Kicker" hint="Static text shown above the headline.">
        <Input
          value={value.kicker ?? ""}
          onChange={(e) => onChange({ ...value, kicker: e.target.value })}
          placeholder="Latest run"
        />
      </Field>
      <div className="grid grid-cols-2 gap-3">
        <SlotField
          label="Status field"
          hint="Drives the accent color."
          placeholder="status"
          fieldKey="statusField"
          value={value}
          onChange={onChange}
          fieldOptions={fieldOptions}
        />
        <SlotField
          label="Status label field"
          hint="Pill text."
          placeholder="status"
          fieldKey="statusLabelField"
          value={value}
          onChange={onChange}
          fieldOptions={fieldOptions}
        />
      </div>
    </Section>
  );
}

function HeadlineSection({ value, onChange, fieldOptions }: SectionProps) {
  const missing = !value.titleField?.trim() && !value.actorNameField?.trim();
  return (
    <Section title="Headline" hint="The main thing this run/record is about.">
      <SlotField
        label="Title field"
        hint="e.g. nodeName, or a commit/PR title from payload."
        placeholder="nodeName"
        fieldKey="titleField"
        value={value}
        onChange={onChange}
        fieldOptions={fieldOptions}
        invalid={missing}
      />
      <div className="grid grid-cols-2 gap-3">
        <SlotField
          label="Link field"
          hint="Optional URL for the title."
          placeholder="payload.data.head_commit.url"
          fieldKey="hrefField"
          value={value}
          onChange={onChange}
          fieldOptions={fieldOptions}
        />
        <SlotField
          label="Subtitle field"
          hint="e.g. repo + branch."
          placeholder="rootEvent.customName"
          fieldKey="subtitleField"
          value={value}
          onChange={onChange}
          fieldOptions={fieldOptions}
        />
      </div>
    </Section>
  );
}

function WhoSection({ value, onChange, fieldOptions }: SectionProps) {
  const missing = !value.titleField?.trim() && !value.actorNameField?.trim();
  return (
    <Section title="Who / what" hint="The person or trigger behind the record. Optional.">
      <div className="grid grid-cols-2 gap-3">
        <SlotField
          label="Name field"
          placeholder="nodeName"
          fieldKey="actorNameField"
          value={value}
          onChange={onChange}
          fieldOptions={fieldOptions}
          invalid={missing}
        />
        <SlotField
          label="Avatar field"
          hint="Image URL."
          placeholder="payload.data.sender.avatar_url"
          fieldKey="actorAvatarField"
          value={value}
          onChange={onChange}
          fieldOptions={fieldOptions}
        />
      </div>
    </Section>
  );
}

function TimingSection({ value, onChange, fieldOptions }: SectionProps) {
  return (
    <Section title="Timing" hint="When it happened and how long it took.">
      <div className="grid grid-cols-2 gap-3">
        <SlotField
          label="Timestamp field"
          hint="Rendered relative."
          placeholder="createdAt"
          fieldKey="timestampField"
          value={value}
          onChange={onChange}
          fieldOptions={fieldOptions}
        />
        <SlotField
          label="Duration field"
          hint="Milliseconds."
          placeholder="durationMs"
          fieldKey="durationField"
          value={value}
          onChange={onChange}
          fieldOptions={fieldOptions}
        />
      </div>
    </Section>
  );
}

function SecondaryPersonSection({ value, onChange, fieldOptions }: SectionProps) {
  return (
    <Section title="Secondary person" hint="An approver, reviewer, owner, or commander. Optional.">
      <Field label="Label" hint="Static text shown before the person.">
        <Input
          value={value.approverLabel ?? ""}
          onChange={(e) => onChange({ ...value, approverLabel: e.target.value })}
          placeholder="Approved by"
        />
      </Field>
      <div className="grid grid-cols-2 gap-3">
        <SlotField
          label="Name field"
          placeholder="payload.data.pull_request.user.login"
          fieldKey="approverNameField"
          value={value}
          onChange={onChange}
          fieldOptions={fieldOptions}
        />
        <SlotField
          label="Avatar field"
          hint="Image URL."
          placeholder="payload.data.pull_request.user.avatar_url"
          fieldKey="approverAvatarField"
          value={value}
          onChange={onChange}
          fieldOptions={fieldOptions}
        />
      </div>
    </Section>
  );
}

/**
 * Source-aware: for runs the checks are the run's stages (its `executions`
 * array); for memory they're an explicit `checks` payload. Executions spotlight
 * a single stage, so there is no array to configure.
 */
function ChecksSection({ value, onChange, fieldOptions }: SectionProps) {
  if (value.dataSource.kind === "executions") {
    return (
      <Section title="Stages">
        <p className="text-[11px] text-slate-400 dark:text-gray-500">
          An executions source spotlights a single stage, so there is no stage strip to configure. Use a runs source to
          show a run&apos;s stages.
        </p>
      </Section>
    );
  }

  const stages = checksAreStages(value.dataSource.kind);
  return (
    <Section
      title={stages ? "Stages" : "Checks"}
      hint={
        stages
          ? "Reads the run's executions array. Status maps RESULT_/STATE_ values automatically."
          : "A field resolving to an array of checks, plus the paths within each item."
      }
    >
      <SlotField
        label={stages ? "Stages array" : "Checks array"}
        hint={stages ? "Dot path to the array, e.g. executions." : "Dot path to the array, e.g. checks."}
        placeholder={stages ? "executions" : "checks"}
        fieldKey="checksField"
        value={value}
        onChange={onChange}
        fieldOptions={fieldOptions}
      />
      <div className="grid grid-cols-2 gap-3">
        <SlotField
          label="Item name path"
          hint={stages ? "e.g. nodeName." : "Within each item."}
          placeholder={stages ? "nodeName" : "name"}
          fieldKey="checkNameField"
          value={value}
          onChange={onChange}
          fieldOptions={fieldOptions}
        />
        <SlotField
          label="Item status path"
          hint={stages ? "e.g. result." : "Within each item."}
          placeholder={stages ? "result" : "status"}
          fieldKey="checkStatusField"
          value={value}
          onChange={onChange}
          fieldOptions={fieldOptions}
        />
      </div>
    </Section>
  );
}

interface SectionProps {
  value: SpotlightPanelContent;
  onChange: (next: SpotlightPanelContent) => void;
  fieldOptions: string[];
}
