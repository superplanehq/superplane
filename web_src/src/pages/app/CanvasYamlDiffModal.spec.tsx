import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import type { MultiFileDiffProps } from "@pierre/diffs/react";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { CanvasYamlDiffModal } from "./CanvasYamlDiffModal";

type TestMultiFileDiffProps = MultiFileDiffProps<never, never>;

const diffProps = vi.hoisted(() => ({
  latest: null as TestMultiFileDiffProps | null,
}));

vi.mock("@pierre/diffs/react", () => ({
  MultiFileDiff: (props: TestMultiFileDiffProps) => {
    diffProps.latest = props;
    return <div data-testid="multi-file-diff" />;
  },
}));

function yamlFileText(file: TestMultiFileDiffProps["oldFile"] | TestMultiFileDiffProps["newFile"]): string | undefined {
  return file?.contents;
}

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

    const latest = diffProps.latest;
    const options = latest && latest.options;
    expect(screen.getByTestId("multi-file-diff")).toBeInTheDocument();
    expect(yamlFileText(latest && latest.oldFile)).toBe("name: old\nshared: same\n");
    expect(yamlFileText(latest && latest.newFile)).toBe("name: new\nshared: same\n");
    expect(options?.lineDiffType).toBe("word");
    expect(options?.parseDiffOptions).toEqual({ context: 6 });
    expect(options?.unsafeCSS).toContain("--diffs-bg-context-override: #ffffff");
    expect(options?.unsafeCSS).toContain('[data-line-type="context"]');
    expect(options?.unsafeCSS).toContain("--diffs-bg-addition-emphasis-override");
    expect(options?.unsafeCSS).toContain("--diffs-bg-deletion-emphasis-override");
  });
});
