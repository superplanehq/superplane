import { CanvasMarkdown, NodeChipContext } from "@/ui/Markdown/CanvasMarkdown";

interface ReportMarkdownProps {
  children: string;
  className?: string;
  nodeRefs?: NodeChipContext;
}

//
// ReportMarkdown is the shared Canvas markdown renderer used inside Reports.
// It is kept as a thin wrapper so existing imports stay stable; new call-sites
// should depend on CanvasMarkdown directly.
//
export function ReportMarkdown({ children, className, nodeRefs }: ReportMarkdownProps) {
  return (
    <CanvasMarkdown className={className} nodeRefs={nodeRefs}>
      {children}
    </CanvasMarkdown>
  );
}
