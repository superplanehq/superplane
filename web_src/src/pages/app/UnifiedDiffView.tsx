import { useMemo } from "react";
import { Diff, Hunk, parseDiff } from "react-diff-view";
import "react-diff-view/style/index.css";
import type { DraftDiffLine } from "./draftNodeDiff";

function toUnifiedDiffText(lines: DraftDiffLine[]): string {
  return lines
    .map((line) => {
      if (line.prefix === "meta") {
        return line.text;
      }

      if (line.prefix === "context") {
        if (line.text.startsWith("@@")) {
          return line.text;
        }

        return ` ${line.text}`;
      }

      return `${line.prefix}${line.text}`;
    })
    .join("\n");
}

export function UnifiedDiffView({
  diffId,
  emptyMessage,
  lines,
}: {
  diffId: string;
  emptyMessage: string;
  lines: DraftDiffLine[];
}) {
  const files = useMemo(() => parseDiff(toUnifiedDiffText(lines), { nearbySequences: "zip" }), [lines]);

  if (!files.length) {
    return <p className="text-xs text-content-secondary">{emptyMessage}</p>;
  }

  return (
    <div className="overflow-hidden rounded-md border border-edge-default bg-surface-raised">
      <div className="max-h-96 overflow-auto">
        {files.map((file) => (
          <Diff
            key={`${diffId}-${file.oldRevision}-${file.newRevision}`}
            viewType="split"
            diffType={file.type}
            hunks={file.hunks}
          >
            {(hunks) => hunks.map((hunk) => <Hunk key={`${diffId}-${hunk.content}`} hunk={hunk} />)}
          </Diff>
        ))}
      </div>
    </div>
  );
}
