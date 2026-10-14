import type { FactoriesFactoryAgentResource } from "@/api-client";
import { SkillPromptTextarea } from "./SkillPromptTextarea";
import { Editor } from "@monaco-editor/react";

import { Textarea } from "@/components/ui/textarea";
import { useTheme } from "@/contexts/useTheme";

import type { PlanningReviewStepKind } from "./planningReviewMockup";

const LINE_HEIGHT = 20;
const MIN_LINES = 2;
const MAX_LINES = 16;

/** Grow with the script, but never far enough to hide the rest of the panel. */
function editorHeight(value: string): number {
  const lines = value === "" ? MIN_LINES : value.split("\n").length;
  return Math.min(Math.max(lines, MIN_LINES), MAX_LINES) * LINE_HEIGHT;
}

/**
 * Bash steps use shell highlighting. Prompts offer skill commands when the
 * workspace skills feature is enabled, and otherwise use a plain textarea.
 *
 * Monaco pads vertically only, so the wrapper supplies the horizontal gutter.
 */
export function PlanningReviewStepBody({
  kind,
  value,
  onChange,
  label,
  testId,
  skills,
  skillHref,
}: {
  skills?: FactoriesFactoryAgentResource[];
  skillHref?: (skill: FactoriesFactoryAgentResource) => string | undefined;
  kind: PlanningReviewStepKind;
  value: string;
  onChange: (value: string) => void;
  label: string;
  testId: string;
}) {
  const { resolvedTheme } = useTheme();

  if (kind === "prompt" && skills) {
    return (
      <SkillPromptTextarea
        skillHref={skillHref}
        value={value}
        onChange={onChange}
        skills={skills}
        label={label}
        testId={testId}
      />
    );
  }

  if (kind === "prompt") {
    return (
      <Textarea
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder="Tell the agent what to do in this step."
        aria-label={label}
        data-testid={testId}
        className="max-h-72 min-h-24 resize-y overflow-auto rounded-lg bg-card px-3 py-2.5 text-[13px] leading-relaxed shadow-none"
      />
    );
  }

  return (
    <div
      className="relative overflow-hidden rounded-lg border border-border bg-card px-3 py-2"
      data-testid={`${testId}-editor`}
    >
      <Editor
        height={editorHeight(value)}
        language="shell"
        value={value}
        theme={resolvedTheme === "dark" ? "vs-dark" : "vs"}
        onChange={(next) => onChange(next ?? "")}
        options={{
          ariaLabel: label,
          minimap: { enabled: false },
          lineNumbers: "off",
          glyphMargin: false,
          folding: false,
          lineDecorationsWidth: 0,
          lineNumbersMinChars: 0,
          fontSize: 13,
          lineHeight: LINE_HEIGHT,
          wordWrap: "on",
          scrollBeyondLastLine: false,
          automaticLayout: true,
          overviewRulerLanes: 0,
          renderLineHighlight: "none",
          padding: { top: 0, bottom: 0 },
          scrollbar: { vertical: "auto", horizontal: "auto", verticalScrollbarSize: 10 },
          tabSize: 2,
        }}
      />
      {value === "" ? (
        <p className="pointer-events-none absolute top-2 left-3 text-[13px] text-muted-foreground">
          Shell commands to run on the runner.
        </p>
      ) : null}
    </div>
  );
}
