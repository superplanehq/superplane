import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createRef } from "react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { ConsoleContextProvider } from "./ConsoleContextProvider";
import { MarkdownPanelEditor } from "./MarkdownPanelEditor";
import { useMarkdownVariables, type MarkdownVariablesResult } from "./useMarkdownVariables";
import { DOLLAR_REWRITE_IDENTIFIER } from "./widget/celExpr";

// Control the resolved/loading state directly so the editor preview's loading
// gate can be exercised without standing up the underlying query machinery.
vi.mock("./useMarkdownVariables", () => ({
  useMarkdownVariables: vi.fn(),
}));

function mockVariables(result: Partial<MarkdownVariablesResult>) {
  vi.mocked(useMarkdownVariables).mockReturnValue({
    vars: {},
    isLoading: false,
    baseLoading: false,
    sideloadLoading: false,
    searchingNames: [],
    errors: [],
    ...result,
  });
}

function renderEditor(draftBody: string) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <ConsoleContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[]} canRunNodes={false}>
          <MarkdownPanelEditor
            panelId="panel-1"
            canvasId="canvas-1"
            draftTitle=""
            setDraftTitle={() => {}}
            draftBody={draftBody}
            setDraftBody={() => {}}
            draftVariables={[{ name: "run", source: { kind: "run", select: "latest" } }]}
            setDraftVariables={() => {}}
            titleInputRef={createRef<HTMLInputElement>()}
            textareaRef={createRef<HTMLTextAreaElement>()}
            onCancel={() => {}}
            onCommit={() => {}}
          />
        </ConsoleContextProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

afterEach(() => {
  vi.clearAllMocks();
});

describe("MarkdownPanelEditor preview loading gate", () => {
  it("shows a spinner instead of empty node fields while run executions side-load", () => {
    // The base run row resolved, but the per-run executions backing
    // `$["Deploy"]` are still loading. The preview must hold the spinner
    // rather than flashing an empty interpolated field — matching the
    // saved read-only view.
    mockVariables({ vars: { run: { status: "passed", $: {} } }, isLoading: true, sideloadLoading: true });
    renderEditor('Output: {{ run.$["Deploy"].data.url }}');

    expect(screen.getByTestId("console-markdown-loading")).toBeTruthy();
    expect(screen.queryByTestId("console-markdown")).toBeNull();
  });

  it("renders the interpolated preview once variables finish loading", () => {
    const deployNodes = { Deploy: { data: { url: "https://example.com/run/42" } } };
    mockVariables({
      vars: { run: { $: deployNodes, [DOLLAR_REWRITE_IDENTIFIER]: deployNodes } },
      isLoading: false,
    });
    renderEditor('Output: {{ run.$["Deploy"].data.url }}');

    expect(screen.queryByTestId("console-markdown-loading")).toBeNull();
    expect(screen.getByTestId("console-markdown").textContent).toMatch(/https:\/\/example\.com\/run\/42/);
  });

  it("does not gate a static body that references no variables", () => {
    mockVariables({ vars: {}, isLoading: true, baseLoading: true });
    renderEditor("This body has no variables.");

    expect(screen.queryByTestId("console-markdown-loading")).toBeNull();
    expect(screen.getByTestId("console-markdown").textContent).toMatch(/This body has no variables\./);
  });
});
