import Editor, { type Monaco } from "@monaco-editor/react";

import { FLEET_MANAGER_VALUES_FILE } from "@/lib/fleetManagerSetup";
import { CopyButton } from "@/ui/CopyButton";

const SETUP_CODE_THEME = "superplane-fleet-setup";
const CODE_MAX_HEIGHT_PX = 320;

const EDITOR_OPTIONS = {
  readOnly: true,
  domReadOnly: true,
  minimap: { enabled: false },
  fontSize: 12,
  lineNumbers: "off" as const,
  wordWrap: "on" as const,
  folding: false,
  scrollBeyondLastLine: false,
  renderWhitespace: "none" as const,
  contextmenu: false,
  scrollbar: {
    vertical: "auto" as const,
    horizontal: "auto" as const,
    useShadows: false,
    verticalScrollbarSize: 8,
    horizontalScrollbarSize: 8,
  },
  padding: { top: 12, bottom: 12 },
  overviewRulerLanes: 0,
  hideCursorInOverviewRuler: true,
  overviewRulerBorder: false,
  guides: { indentation: false },
  renderLineHighlight: "none" as const,
  automaticLayout: true,
};

function commandHeight(code: string): number {
  const lines = Math.max(code.split("\n").length, 1);
  // One extra row covers a single wrapped line so the last command stays visible.
  return Math.min(lines * 19 + 16 + 19, CODE_MAX_HEIGHT_PX);
}

function defineSetupCodeTheme(monaco: Monaco) {
  monaco.editor.defineTheme(SETUP_CODE_THEME, {
    base: "vs-dark",
    inherit: true,
    rules: [
      { token: "comment", foreground: "9b9993" },
      { token: "string", foreground: "d6c7a1" },
      { token: "number", foreground: "d6c7a1" },
    ],
    colors: {
      "editor.background": "#201f1a",
      "editor.foreground": "#eeede9",
      "editorCursor.foreground": "#eeede9",
      "editor.selectionBackground": "#34322b",
      "scrollbarSlider.background": "#4a483f",
      "scrollbarSlider.hoverBackground": "#6b6860",
      "scrollbarSlider.activeBackground": "#9b9993",
      "scrollbar.shadow": "#201f1a",
    },
  });
}

const DARK_SCROLLBAR =
  "[&_.scrollbar]:!bg-transparent [&_.slider]:!bg-[#4a483f] [&_.slider:hover]:!bg-[#9b9993] [&_.decorationsOverviewRuler]:!hidden";

function SetupCodeBlock({
  code,
  language,
  copyLabel,
}: {
  code: string;
  language: "yaml" | "shell";
  copyLabel: string;
}) {
  const height = commandHeight(code);
  return (
    <div
      className={`fleet-setup-code overflow-hidden rounded-md border border-[#34322b] bg-[#201f1a] [&_.cursor]:hidden ${DARK_SCROLLBAR}`}
      data-language={language}
    >
      <div className="flex justify-end border-b border-[#34322b] px-2 py-1">
        <CopyButton variant="icon" dark ariaLabel={copyLabel} copiedAriaLabel="Copied" text={code} />
      </div>
      <Editor
        height={height}
        language={language}
        value={code}
        theme={SETUP_CODE_THEME}
        beforeMount={defineSetupCodeTheme}
        options={EDITOR_OPTIONS}
        loading={
          <pre className={`overflow-auto p-4 font-mono text-xs text-[#eeede9] ${DARK_SCROLLBAR}`}>
            <code>{code}</code>
          </pre>
        }
      />
    </div>
  );
}

export function FleetManagerCommandBlock({
  code,
  language,
  copyLabel,
}: {
  code: string;
  language: "yaml" | "shell";
  copyLabel: string;
}) {
  return <SetupCodeBlock code={code} language={language} copyLabel={copyLabel} />;
}

export function FleetManagerInstallCommands({
  valuesCommand,
  helmCommand,
}: {
  valuesCommand: string;
  helmCommand: string;
}) {
  return (
    <div className="space-y-5" data-testid="fleet-manager-install-command">
      <section className="space-y-2" aria-labelledby="fleet-values-command">
        <h3 id="fleet-values-command" className="text-sm font-medium">
          1. Create {FLEET_MANAGER_VALUES_FILE}
        </h3>
        <p className="text-xs leading-5 text-muted-foreground">
          Run this command. It writes {FLEET_MANAGER_VALUES_FILE}.
        </p>
        <FleetManagerCommandBlock code={valuesCommand} language="shell" copyLabel="Copy values command" />
      </section>
      <section className="space-y-2" aria-labelledby="fleet-install-command">
        <h3 id="fleet-install-command" className="text-sm font-medium">
          2. Install the chart
        </h3>
        <p className="text-xs leading-5 text-muted-foreground">
          Run this command in the same directory. It reads {FLEET_MANAGER_VALUES_FILE}.
        </p>
        <FleetManagerCommandBlock code={helmCommand} language="shell" copyLabel="Copy install command" />
      </section>
    </div>
  );
}
