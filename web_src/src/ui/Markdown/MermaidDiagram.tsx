import { useEffect, useId, useRef, useState } from "react";
import { AlertTriangle } from "lucide-react";

//
// Mermaid diagrams rendered from ```mermaid fenced code blocks.
//
// Mermaid is loaded lazily on first use — it pulls in a few hundred KB of
// diagram renderers we don't want on every page. Each diagram renders to an
// SVG string via `mermaid.render`, which we inject into a container div.
//
// Security: we run mermaid with `securityLevel: "strict"`, which routes
// user-authored labels through its HTML sanitizer and forbids raw HTML in
// node labels. The resulting SVG never contains `<script>` or `on*=`
// handlers, so inserting it via `dangerouslySetInnerHTML` is safe.
//

let mermaidInitPromise: Promise<typeof import("mermaid").default> | null = null;

async function getMermaid() {
  if (!mermaidInitPromise) {
    mermaidInitPromise = import("mermaid").then((mod) => {
      const mermaid = mod.default;
      mermaid.initialize({
        startOnLoad: false,
        securityLevel: "strict",
        theme: "neutral",
        fontFamily: "inherit",
      });
      return mermaid;
    });
  }
  return mermaidInitPromise;
}

interface MermaidDiagramProps {
  code: string;
}

export function MermaidDiagram({ code }: MermaidDiagramProps) {
  const reactId = useId();
  // Mermaid requires an id that starts with a letter and has no colons.
  const diagramId = `mermaid-${reactId.replace(/[^a-zA-Z0-9]/g, "")}`;
  const hostRef = useRef<HTMLDivElement | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isRendered, setIsRendered] = useState(false);

  useEffect(() => {
    let cancelled = false;

    (async () => {
      try {
        const mermaid = await getMermaid();
        const { svg, bindFunctions } = await mermaid.render(diagramId, code);

        if (cancelled || !hostRef.current) {
          return;
        }

        hostRef.current.innerHTML = svg;
        if (bindFunctions) {
          bindFunctions(hostRef.current);
        }
        setError(null);
        setIsRendered(true);
      } catch (err) {
        if (cancelled) return;
        const message = err instanceof Error ? err.message : String(err);
        setError(message);
        setIsRendered(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [code, diagramId]);

  if (error) {
    return (
      <div className="my-2 rounded-md border border-red-200 bg-red-50 p-3 text-xs text-red-700">
        <div className="mb-1 flex items-center gap-1.5 font-semibold">
          <AlertTriangle className="h-3.5 w-3.5" />
          Mermaid diagram failed to render
        </div>
        <pre className="whitespace-pre-wrap break-words text-[11px] text-red-600">{error}</pre>
        <details className="mt-2">
          <summary className="cursor-pointer text-[11px] text-red-500">Show source</summary>
          <pre className="mt-1 overflow-x-auto rounded bg-white p-2 text-[11px] text-gray-700">{code}</pre>
        </details>
      </div>
    );
  }

  return (
    <div
      ref={hostRef}
      data-testid="mermaid-diagram"
      className="my-2 flex justify-center overflow-x-auto rounded-md border border-slate-200 bg-white p-3 [&>svg]:max-w-full [&>svg]:h-auto"
      aria-busy={!isRendered}
    />
  );
}
