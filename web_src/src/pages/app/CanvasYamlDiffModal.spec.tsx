import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { MultiFileDiffProps } from "@pierre/diffs/react";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { CanvasYamlDiffModal } from "./CanvasYamlDiffModal";

type TestMultiFileDiffProps = MultiFileDiffProps<never>;

const diffProps = vi.hoisted(() => ({
  latest: null as TestMultiFileDiffProps | null,
}));

vi.mock("@pierre/diffs/react", () => ({
  MultiFileDiff: (props: TestMultiFileDiffProps) => {
    diffProps.latest = props;
    return <div data-testid="multi-file-diff" />;
  },
}));

describe("CanvasYamlDiffModal", () => {
  beforeEach(() => {
    diffProps.latest = null;
  });

  it("renders YAML diffs with neutral context rows and inline word highlights", () => {
    render(
      <ThemeProvider>
        <CanvasYamlDiffModal
          open
          onOpenChange={() => undefined}
          liveYamlText={"name: old\nshared: same\n"}
          draftYamlText={"name: new\nshared: same\n"}
          filename="canvas.yaml"
        />
      </ThemeProvider>,
    );

    expect(screen.getByTestId("multi-file-diff")).toBeInTheDocument();
    expect(diffProps.latest?.oldFile.contents).toBe("name: old\nshared: same\n");
    expect(diffProps.latest?.newFile.contents).toBe("name: new\nshared: same\n");
    expect(diffProps.latest?.options?.lineDiffType).toBe("word");
    expect(diffProps.latest?.options?.parseDiffOptions).toEqual({ context: 6 });
    expect(diffProps.latest?.options?.unsafeCSS).toContain("--diffs-bg-context-override: #ffffff");
    expect(diffProps.latest?.options?.unsafeCSS).toContain('[data-line-type="context"]');
    expect(diffProps.latest?.options?.unsafeCSS).toContain("--diffs-bg-addition-emphasis-override");
    expect(diffProps.latest?.options?.unsafeCSS).toContain("--diffs-bg-deletion-emphasis-override");
  });
});
