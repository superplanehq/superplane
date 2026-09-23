import { type Dispatch, type SetStateAction } from "react";

import { IntakeEventRow } from "./IntakeEventRow";
import {
  INTAKE_SETTINGS_COPY,
  intakeSettingsSectionDomId,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";

const SENTRY_INTAKE_EVENTS = [
  {
    key: "sentryNewIssues",
    title: "A new issue is created",
    description: "Sentry adds an issue.",
    testId: "sentry-intake-new-issues",
  },
  {
    key: "sentryRegressedIssues",
    title: "An issue becomes unresolved",
    description: "Sentry marks a resolved issue as unresolved.",
    testId: "sentry-intake-regressed-issues",
  },
  {
    key: "sentryAssignedIssues",
    title: "An issue is assigned",
    description: "A person assigns the issue.",
    testId: "sentry-intake-assigned-issues",
  },
] as const satisfies ReadonlyArray<{
  key: "sentryNewIssues" | "sentryRegressedIssues" | "sentryAssignedIssues";
  title: string;
  description: string;
  testId: string;
}>;

export function SentryIntakeFilterFields({
  sourceId,
  settings,
  onSettingsChange,
}: {
  sourceId: LineIntakeSourceId;
  settings: IntakeSourceSettings;
  onSettingsChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  /** Kept for callers that still pass a layout. The event list is always a stack. */
  layout?: "stack" | "grid";
}) {
  if (sourceId !== "sentry-exceptions") {
    return null;
  }

  function update<K extends keyof IntakeSourceSettings>(key: K, value: IntakeSourceSettings[K]) {
    onSettingsChange((current) => ({ ...current, [key]: value }));
  }

  return (
    <div className="flex flex-col gap-6">
      <section id={intakeSettingsSectionDomId("triggers")} className="scroll-mt-6 min-w-0">
        <h3 className="workspace-section-title">{INTAKE_SETTINGS_COPY.eventsThatCreateTasks}</h3>
        <p className="mt-1 text-[13px] text-muted-foreground">{INTAKE_SETTINGS_COPY.eventsThatCreateTasksHelper}</p>
        <div
          className="mt-3 divide-y divide-border overflow-hidden rounded-lg border border-border"
          role="group"
          aria-label={INTAKE_SETTINGS_COPY.eventsThatCreateTasks}
        >
          {SENTRY_INTAKE_EVENTS.map((event) => (
            <IntakeEventRow
              key={event.key}
              title={event.title}
              description={event.description}
              active={settings[event.key]}
              onToggle={() => update(event.key, !settings[event.key])}
              testId={event.testId}
            />
          ))}
        </div>
      </section>
    </div>
  );
}
