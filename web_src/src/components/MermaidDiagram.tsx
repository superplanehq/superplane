import mermaid from "mermaid";
import { useEffect, useRef, useState } from "react";

mermaid.initialize({
  startOnLoad: false,
  theme: "neutral",
  flowchart: { curve: "basis", padding: 12 },
  securityLevel: "loose",
});

let idCounter = 0;

export type MermaidDiagramProps = {
  definition: string;
  className?: string;
};

export function MermaidDiagram({ definition, className }: MermaidDiagramProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const container = containerRef.current;
    if (!container || !definition.trim()) {
      return;
    }

    let cancelled = false;
    const renderingId = `mermaid-${++idCounter}`;

    void (async () => {
      try {
        const { svg } = await mermaid.render(renderingId, definition);
        if (!cancelled && container) {
          container.innerHTML = svg;
          setError(null);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to render diagram");
          container.innerHTML = "";
        }
        const orphan = document.getElementById("d" + renderingId);
        orphan?.remove();
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [definition]);

  if (error) {
    return (
      <div className="my-1 rounded-md bg-slate-50 text-xs">
        <div className="px-2 py-1.5 text-muted-foreground italic">Could not render diagram.</div>
        <pre className="overflow-auto border-t border-slate-200 px-2 py-2 text-[10px] leading-relaxed text-slate-500">
          <code>{definition}</code>
        </pre>
      </div>
    );
  }

  return <div ref={containerRef} className={className} />;
}
