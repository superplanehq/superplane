import { useCallback, useEffect, useState } from "react";
import { parseCurlCommand, type CurlParseResult } from "@/lib/curlParser";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import type { CustomFieldRendererContext, NodeInfo } from "@/pages/workflowv2/mappers/types";

type HttpCurlCustomFieldProps = {
  node: NodeInfo;
  context?: CustomFieldRendererContext;
};

export function HttpCurlCustomField({ node: _node, context: _context }: HttpCurlCustomFieldProps) {
  const [curlInput, setCurlInput] = useState("");
  const [parseResult, setParseResult] = useState<CurlParseResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isParsing, setIsParsing] = useState(false);

  useEffect(() => {
    if (!curlInput.trim()) {
      setParseResult(null);
      setError(null);
      return;
    }

    const timer = setTimeout(() => {
      setIsParsing(true);
      try {
        const result = parseCurlCommand(curlInput);
        setParseResult(result);
        setError(result.error ?? null);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to parse curl command");
        setParseResult(null);
      } finally {
        setIsParsing(false);
      }
    }, 500);

    return () => clearTimeout(timer);
  }, [curlInput]);

  const handleAutoPopulate = useCallback(() => {
    // Placeholder until node configuration wiring is implemented.
  }, []);

  return (
    <div className="mt-4 rounded-md border bg-white">
      <div className="border-b px-4 py-3">
        <h3 className="flex items-center gap-2 text-sm font-medium">
          <span>Import from cURL</span>
          <Badge variant="secondary" className="text-xs">
            Beta
          </Badge>
        </h3>
      </div>
      <div className="space-y-4 p-4">
        <div>
          <Label htmlFor="curl-input">Paste curl command</Label>
          <Textarea
            id="curl-input"
            placeholder={`curl -X POST https://api.example.com/users -H "Content-Type: application/json" -d '{"name":"John"}'`}
            className="mt-2 font-mono text-xs"
            value={curlInput}
            onChange={(event) => setCurlInput(event.target.value)}
            disabled={isParsing}
          />
          <p className="mt-1 text-xs text-gray-500">
            Paste your curl command here to automatically populate HTTP configuration fields.
          </p>
        </div>

        {isParsing && (
          <div className="flex items-center gap-2 text-sm text-gray-500">
            <div className="h-4 w-4 animate-spin rounded-full border-b-2 border-gray-500" />
            <span>Parsing curl command...</span>
          </div>
        )}

        {error && (
          <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</div>
        )}

        {parseResult && !parseResult.error && (
          <div className="space-y-3">
            <div className="flex items-start justify-between">
              <div>
                <h4 className="font-medium">Parsed Result</h4>
                <p className="mt-1 text-sm text-gray-600">
                  Method: <span className="rounded bg-gray-100 px-1 font-mono">{parseResult.method}</span> | URL:{" "}
                  <span className="max-w-xs truncate rounded bg-gray-100 px-1 font-mono">{parseResult.url}</span>
                </p>
              </div>
              <Button size="sm" onClick={handleAutoPopulate} variant="outline" className="text-xs">
                Auto-populate
              </Button>
            </div>

            {parseResult.headers.length > 0 && (
              <div>
                <h4 className="text-sm font-medium">Headers</h4>
                <div className="mt-1 flex flex-wrap gap-1">
                  {parseResult.headers.map((header) => (
                    <Badge key={`${header.name}-${header.value}`} variant="secondary" className="text-xs">
                      {header.name}: {header.value}
                    </Badge>
                  ))}
                </div>
              </div>
            )}

            {parseResult.queryParams.length > 0 && (
              <div>
                <h4 className="text-sm font-medium">Query Params</h4>
                <div className="mt-1 flex flex-wrap gap-1">
                  {parseResult.queryParams.map((param) => (
                    <Badge key={`${param.key}-${param.value}`} variant="secondary" className="text-xs">
                      {param.key}: {param.value}
                    </Badge>
                  ))}
                </div>
              </div>
            )}

            {parseResult.body && (
              <div>
                <h4 className="text-sm font-medium">Body</h4>
                <p className="mt-1 max-h-20 overflow-auto rounded bg-gray-100 p-2 font-mono text-xs">
                  {parseResult.body}
                </p>
              </div>
            )}
          </div>
        )}

        <div className="pt-2">
          <p className="text-xs text-gray-500">
            <strong>Supported curl formats:</strong> curl -X POST https://api.example.com/users -H "Content-Type:
            application/json" -d '{"{"}name":"John"{"}"}'
          </p>
        </div>
      </div>
    </div>
  );
}
