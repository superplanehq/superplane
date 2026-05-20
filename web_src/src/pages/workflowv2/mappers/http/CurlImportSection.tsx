import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { parseCurl } from "@/lib/parseCurl";
import { curlToHttpConfig } from "@/lib/curlToHttpConfig";

interface Props {
  onApply: (patch: Record<string, unknown>) => void;
  disabled?: boolean;
}

export function CurlImportSection({ onApply, disabled }: Props) {
  const [text, setText] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [hint, setHint] = useState<string | null>(null);

  const trimmed = text.trim();
  const looksLikeCurl = /^curl\b/.test(trimmed);
  const canParse = looksLikeCurl && !disabled;

  const handleParse = () => {
    setError(null);
    setHint(null);

    if (!looksLikeCurl) {
      setError("Paste a command that starts with `curl`.");
      return;
    }

    try {
      const parsed = parseCurl(trimmed);
      if (!parsed.url) {
        setError("Couldn't find a URL in the command.");
        return;
      }

      const { patch, authNeedsSecret } = curlToHttpConfig(parsed);
      onApply(patch as Record<string, unknown>);

      if (authNeedsSecret) {
        setHint("Authorization detected — point the credential field at a secret to finish.");
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to parse curl command.");
    }
  };

  return (
    <div
      data-testid="curl-import-section"
      className="rounded-md border border-dashed border-gray-300 dark:border-gray-700 p-3 mb-2 space-y-2"
    >
      <p className="text-sm font-medium text-gray-700 dark:text-gray-200">Prefill from a curl command</p>

      <Textarea
        data-testid="curl-import-textarea"
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={(e) => {
          // Enter parses & fills. Shift+Enter inserts a newline for manual editing.
          if (e.key === "Enter" && !e.shiftKey) {
            e.preventDefault();
            if (canParse) handleParse();
          }
        }}
        placeholder={`curl -X POST 'https://api.example.com/x' \\\n  -H 'Authorization: Bearer …' \\\n  --json '{"key":"value"}'`}
        rows={6}
        spellCheck={false}
        className="font-mono text-xs"
        disabled={disabled}
      />

      <div className="flex items-center justify-between gap-2">
        <p className="text-xs text-gray-500">
          Press <kbd className="px-1 py-0.5 rounded border text-[10px]">Enter</kbd> to parse;{" "}
          <kbd className="px-1 py-0.5 rounded border text-[10px]">Shift+Enter</kbd> inserts a newline. Secret values must
          be wired manually.
        </p>
        <Button
          data-testid="curl-import-apply"
          variant="default"
          size="sm"
          onClick={handleParse}
          disabled={!canParse}
        >
          Parse &amp; fill
        </Button>
      </div>

      {error && (
        <p className="text-xs text-red-600 dark:text-red-400" role="alert">
          {error}
        </p>
      )}
      {hint && !error && <p className="text-xs text-amber-600 dark:text-amber-400">{hint}</p>}
    </div>
  );
}
