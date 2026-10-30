import Editor, { type Monaco } from "@monaco-editor/react";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
  const visualLines = code.split("\n").reduce((count, line) => count + Math.max(1, Math.ceil(line.length / 56)), 0);
  return Math.min(Math.max(visualLines, 1) * 19 + 24, CODE_MAX_HEIGHT_PX);
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

function FleetManagerKubeContextField({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor="fleet-manager-kube-context">Kubernetes context</Label>
      <Input
        id="fleet-manager-kube-context"
        value={value}
        onChange={(event) => onChange(event.target.value.replace(/[\r\n]/g, ""))}
        autoComplete="off"
        spellCheck={false}
        aria-describedby="fleet-manager-kube-context-help"
      />
      <p id="fleet-manager-kube-context-help" className="text-xs leading-5 text-muted-foreground">
        Run kubectl config current-context. The install command uses this context.
      </p>
    </div>
  );
}

export function FleetManagerHelmInstall({
  kubeContext,
  onKubeContextChange,
  helmCommand,
}: {
  kubeContext: string;
  onKubeContextChange: (value: string) => void;
  helmCommand: string | null;
}) {
  return (
    <div className="space-y-3">
      <FleetManagerKubeContextField value={kubeContext} onChange={onKubeContextChange} />
      {helmCommand ? (
        <FleetManagerCommandBlock code={helmCommand} language="shell" copyLabel="Copy install command" />
      ) : (
        <p className="text-sm text-muted-foreground">Enter the Kubernetes context to build the install command.</p>
      )}
    </div>
  );
}

export function FleetManagerInstallCommands({
  valuesCommand,
  helmCommand,
  kubeContext,
  onKubeContextChange,
}: {
  valuesCommand: string;
  helmCommand: string | null;
  kubeContext: string;
  onKubeContextChange: (value: string) => void;
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
        {helmCommand ? (
          <p className="text-xs leading-5 text-muted-foreground">
            Run this command in the same directory. It reads {FLEET_MANAGER_VALUES_FILE}.
          </p>
        ) : null}
        <FleetManagerHelmInstall
          kubeContext={kubeContext}
          onKubeContextChange={onKubeContextChange}
          helmCommand={helmCommand}
        />
      </section>
    </div>
  );
}
