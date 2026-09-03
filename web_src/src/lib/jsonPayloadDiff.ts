/**
 * Minimal line-level diff between two texts, computed via a longest-common-subsequence
 * table. Used to render "the difference against the original" next to the replay
 * payload editor without pulling in a new dependency (Monaco's DiffEditor has no
 * onChange hook, react-diff-view needs a full unified-diff string, and @pierre/diffs
 * is a heavier async/worker API).
 */
export type PayloadDiffLine = {
  type: "unchanged" | "added" | "removed";
  text: string;
};

/** lcs[i][j] = length of the longest common subsequence of a[i:] and b[j:]. */
function buildLcsTable(a: string[], b: string[]): number[][] {
  const n = a.length;
  const m = b.length;
  const lcs: number[][] = Array.from({ length: n + 1 }, () => new Array<number>(m + 1).fill(0));

  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      lcs[i][j] = a[i] === b[j] ? lcs[i + 1][j + 1] + 1 : Math.max(lcs[i + 1][j], lcs[i][j + 1]);
    }
  }

  return lcs;
}

function backtrackDiff(a: string[], b: string[], lcs: number[][]): PayloadDiffLine[] {
  const result: PayloadDiffLine[] = [];
  let i = 0;
  let j = 0;

  while (i < a.length && j < b.length) {
    if (a[i] === b[j]) {
      result.push({ type: "unchanged", text: a[i] });
      i++;
      j++;
    } else if (lcs[i + 1][j] >= lcs[i][j + 1]) {
      result.push({ type: "removed", text: a[i] });
      i++;
    } else {
      result.push({ type: "added", text: b[j] });
      j++;
    }
  }
  while (i < a.length) {
    result.push({ type: "removed", text: a[i] });
    i++;
  }
  while (j < b.length) {
    result.push({ type: "added", text: b[j] });
    j++;
  }

  return result;
}

export function computeLineDiff(original: string, modified: string): PayloadDiffLine[] {
  const originalLines = original.split("\n");
  const modifiedLines = modified.split("\n");
  const lcs = buildLcsTable(originalLines, modifiedLines);
  return backtrackDiff(originalLines, modifiedLines, lcs);
}
