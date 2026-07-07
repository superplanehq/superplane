import { Check, Copy, Maximize2 } from "lucide-react";
import { memo, useCallback, useState } from "react";
import Editor from "@monaco-editor/react";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useTheme } from "@/contexts/useTheme";

interface CodeBlockWidgetProps {
  code: string;
  language?: string;
}

const MONACO_OPTIONS = {
  readOnly: true,
  minimap: { enabled: false },
  fontSize: 12,
  lineNumbers: "off" as const,
  wordWrap: "on" as const,
  folding: true,
  scrollBeyondLastLine: false,
  renderWhitespace: "none" as const,
  contextmenu: false,
  cursorStyle: "line" as const,
  scrollbar: {
    vertical: "auto" as const,
    horizontal: "auto" as const,
  },
  padding: { top: 8, bottom: 8 },
  overviewRulerLanes: 0,
  hideCursorInOverviewRuler: true,
  overviewRulerBorder: false,
  guides: { indentation: false },
  renderLineHighlight: "none" as const,
};

function mapLanguage(lang?: string): string {
  const map: Record<string, string> = {
    yml: "yaml",
    sh: "shell",
    bash: "shell",
    zsh: "shell",
    js: "javascript",
    ts: "typescript",
    py: "python",
    rb: "ruby",
    dockerfile: "dockerfile",
    tf: "hcl",
  };
  if (!lang) return "plaintext";
  return map[lang.toLowerCase()] || lang.toLowerCase();
}

function calcHeight(code: string, maxPx = 250): number {
  const lineCount = code.split("\n").length;
  const lineHeight = 19;
  return Math.min(Math.max(lineCount * lineHeight + 16, 60), maxPx);
}

export const CodeBlockWidget = memo(function CodeBlockWidget({ code, language }: CodeBlockWidgetProps) {
  const [copied, setCopied] = useState(false);
  const [expanded, setExpanded] = useState(false);
  const { resolvedTheme } = useTheme();
  const monacoLang = mapLanguage(language);
  const monacoTheme = resolvedTheme === "dark" ? "vs-dark" : "vs";

  const handleCopy = useCallback(async () => {
    await navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, [code]);

  const height = calcHeight(code);

  return (
    <>
      <div className="group my-4 w-full min-w-0 overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-gray-700 dark:bg-gray-800">
        <div className="flex items-center justify-between border-b border-slate-200 bg-slate-50 px-3 py-1 dark:border-gray-700 dark:bg-gray-900/60">
          <span className="text-[10px] font-medium uppercase tracking-wider text-slate-500 dark:text-gray-400">
            {language || "code"}
          </span>
          <div className="flex items-center gap-1">
            <button
              type="button"
              onClick={handleCopy}
              className="cursor-pointer rounded p-1 text-slate-400 transition-colors hover:bg-slate-200/60 hover:text-slate-600 dark:text-gray-400 dark:hover:bg-gray-700 dark:hover:text-gray-200"
              aria-label="Copy code"
            >
              {copied ? <Check className="size-3.5 text-green-600" /> : <Copy className="size-3.5" />}
            </button>
            <button
              type="button"
              onClick={() => setExpanded(true)}
              className="cursor-pointer rounded p-1 text-slate-400 transition-colors hover:bg-slate-200/60 hover:text-slate-600 dark:text-gray-400 dark:hover:bg-gray-700 dark:hover:text-gray-200"
              aria-label="Expand code"
            >
              <Maximize2 className="size-3.5" />
            </button>
          </div>
        </div>
        <div style={{ height: `${height}px` }}>
          <Editor
            height="100%"
            width="100%"
            defaultLanguage={monacoLang}
            value={code}
            theme={monacoTheme}
            options={MONACO_OPTIONS}
          />
        </div>
      </div>

      <Dialog open={expanded} onOpenChange={setExpanded}>
        <DialogContent size="large" className="w-[90vw] max-h-[85vh] flex flex-col">
          <DialogHeader>
            <DialogTitle className="flex items-center justify-between">
              <span className="text-sm font-medium">{language || "Code"}</span>
              <button
                type="button"
                onClick={handleCopy}
                className="cursor-pointer rounded p-1.5 text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-200"
                aria-label="Copy code"
              >
                {copied ? <Check className="size-4 text-green-600" /> : <Copy className="size-4" />}
              </button>
            </DialogTitle>
          </DialogHeader>
          <div className="min-h-0 flex-1 overflow-hidden rounded-lg border border-slate-200 dark:border-gray-700">
            <Editor
              height={`${Math.min(Math.max(code.split("\n").length * 19 + 20, 200), window.innerHeight * 0.7)}px`}
              width="100%"
              defaultLanguage={monacoLang}
              value={code}
              theme={monacoTheme}
              options={{ ...MONACO_OPTIONS, lineNumbers: "on", fontSize: 13 }}
            />
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
});
