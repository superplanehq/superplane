import { useEffect, useId, useRef, useState } from "react";
import mermaid from "mermaid";

mermaid.initialize({
  startOnLoad: false,
  theme: "base",
  securityLevel: "strict",
  fontFamily: "ui-sans-serif, system-ui, sans-serif",
  themeVariables: {
    // Primary colors — violet palette
    primaryColor: "#ede9fe",
    primaryTextColor: "#4c1d95",
    primaryBorderColor: "#8b5cf6",

    // Secondary — cyan
    secondaryColor: "#ecfeff",
    secondaryTextColor: "#164e63",
    secondaryBorderColor: "#06b6d4",

    // Tertiary — amber
    tertiaryColor: "#fffbeb",
    tertiaryTextColor: "#78350f",
    tertiaryBorderColor: "#f59e0b",

    // Lines and text
    lineColor: "#94a3b8",
    textColor: "#334155",

    // Nodes
    nodeBorder: "#8b5cf6",
    nodeTextColor: "#1e293b",

    // Flowchart
    clusterBkg: "#f8fafc",
    clusterBorder: "#e2e8f0",
    defaultLinkColor: "#8b5cf6",

    // Fonts
    fontSize: "13px",
  },
});

interface MermaidWidgetProps {
  content: string;
}

export function MermaidWidget({ content }: MermaidWidgetProps) {
  const id = useId().replace(/:/g, "m");
  const containerRef = useRef<HTMLDivElement>(null);
  const [svg, setSvg] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function render() {
      try {
        const { svg: rendered } = await mermaid.render(`mermaid-${id}`, content.trim());
        if (!cancelled) {
          setSvg(rendered);
          setError(null);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to render diagram");
          setSvg(null);
        }
        // Clean up mermaid's error container
        document.getElementById(`dmermaid-${id}`)?.remove();
      }
    }

    render();
    return () => {
      cancelled = true;
    };
  }, [content, id]);

  if (error) {
    return (
      <div className="my-2 border border-red-200 bg-red-50 rounded-lg p-2">
        <p className="text-xs text-red-600 font-medium">Diagram error</p>
        <pre className="text-xs text-red-500 mt-1 whitespace-pre-wrap">{content.trim()}</pre>
      </div>
    );
  }

  if (!svg) {
    return (
      <div className="my-2 flex items-center justify-center py-4 text-xs text-slate-400">
        Rendering diagram...
      </div>
    );
  }

  return (
    <div
      ref={containerRef}
      className="my-2 overflow-x-auto rounded-lg border border-slate-200 bg-slate-50/50 p-3 [&_svg]:max-w-full [&_svg]:h-auto [&_svg]:mx-auto"
      dangerouslySetInnerHTML={{ __html: svg }}
    />
  );
}
