import { useMemo } from "react";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { getMockStepConfig } from "./runStepConfigMock";

const noop = () => {};

/**
 * Read-only rendition of the node editing form. It renders the same field controls
 * as the edit sidebar (`ConfigurationFieldRenderer`) but non-interactive
 * (`pointer-events-none` + no-op `onChange`), so the configuration reads exactly like
 * the edit form. Schema/values come from the prototype mock catalog (keyed by
 * component) merged over the node's real configuration, since the run panel has no
 * live catalog access. Rendered inside the timeline's Runtime Config card.
 */
export function RunStepConfigFields({
  component,
  configuration,
}: {
  component?: string;
  configuration?: Record<string, unknown>;
}) {
  const { fields, values } = useMemo(() => {
    const mock = getMockStepConfig(component);
    return { fields: mock.fields, values: { ...mock.values, ...(configuration ?? {}) } };
  }, [component, configuration]);

  const visibleFields = fields.filter((field) => field.name && field.name !== "customName");

  if (visibleFields.length === 0) {
    return <p className="text-[13px] text-slate-400">No configuration for this step.</p>;
  }

  return (
    <div className="pointer-events-none space-y-4" aria-disabled="true">
      {visibleFields.map((field) => (
        <ConfigurationFieldRenderer
          key={field.name}
          field={field}
          value={values[field.name!]}
          onChange={noop}
          allValues={values}
          allowExpressions={false}
        />
      ))}
    </div>
  );
}
