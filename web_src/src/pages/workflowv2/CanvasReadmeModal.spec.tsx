import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

//
// Render CanvasMarkdown as a plain <pre> so assertions can match on the raw
// markdown text without depending on the real remark/rehype pipeline, and so
// node-chip resolution is observable through the `nodeRefs` prop.
//
vi.mock("@/ui/Markdown/CanvasMarkdown", () => ({
  CanvasMarkdown: ({
    children,
    nodeRefs,
  }: {
    children: string;
    nodeRefs?: { nodes: Record<string, string>; linkFor: (slug: string) => string };
  }) => (
    <pre
      data-testid="canvas-markdown"
      data-node-slugs={nodeRefs ? Object.keys(nodeRefs.nodes).join(",") : ""}
      data-node-link-sample={nodeRefs ? nodeRefs.linkFor("api") : ""}
    >
      {children}
    </pre>
  ),
}));

//
// Shim the heavier UI primitives so we can focus on CanvasReadmeModal's own
// logic. Each mock mirrors just enough of the real API surface for the
// component under test.
//
vi.mock("@/components/ui/dialog", () => {
  const Dialog = ({
    children,
    open,
    onOpenChange,
  }: {
    children: ReactNode;
    open: boolean;
    onOpenChange?: (open: boolean) => void;
  }) =>
    open ? (
      <div data-testid="dialog">
        <button data-testid="dialog-close" type="button" onClick={() => onOpenChange?.(false)}>
          close dialog
        </button>
        {children}
      </div>
    ) : null;
  const passthrough = ({ children }: { children?: ReactNode }) => <div>{children}</div>;
  return {
    Dialog,
    DialogContent: passthrough,
    DialogHeader: passthrough,
    DialogTitle: passthrough,
    DialogDescription: passthrough,
    DialogFooter: passthrough,
  };
});

import { CanvasReadmeModal, type CanvasReadmeModalProps } from "./CanvasReadmeModal";

const noop = async () => {};

function renderModal(props: Partial<CanvasReadmeModalProps> = {}) {
  const defaults: CanvasReadmeModalProps = {
    open: true,
    onOpenChange: vi.fn(),
    mode: "live",
    changeManagementEnabled: false,
    liveContent: "# Live readme\nPublished content.",
    draftContent: "",
    isLoadingLive: false,
    isLoadingDraft: false,
    isSavingDraft: false,
    isCreatingChangeRequest: false,
    nodes: { api: "API", "health-check": "Health Check" },
    linkFor: (slug: string) => `/org/canvases/c?node=${slug}`,
    onSaveDraft: vi.fn(noop),
    onCreateChangeRequest: vi.fn(noop),
  };
  const merged: CanvasReadmeModalProps = { ...defaults, ...props };
  const utils = render(<CanvasReadmeModal {...merged} />);
  return { ...utils, props: merged };
}

describe("CanvasReadmeModal", () => {
  const originalConfirm = window.confirm;

  beforeEach(() => {
    window.confirm = originalConfirm;
  });

  it("renders the live readme body when mode is 'live'", () => {
    renderModal({ mode: "live", liveContent: "# Live readme\nHello." });

    const body = screen.getByTestId("canvas-markdown");
    expect(body.textContent).toContain("Live readme");
    expect(screen.queryByLabelText("Canvas readme markdown editor")).toBeNull();
    expect(screen.queryByRole("button", { name: /save draft/i })).toBeNull();
  });

  it("renders the edit body with textarea and footer when mode is 'edit'", () => {
    renderModal({ mode: "edit", draftContent: "# Draft\nWork in progress." });

    const textarea = screen.getByLabelText("Canvas readme markdown editor") as HTMLTextAreaElement;
    expect(textarea).toBeTruthy();
    expect(textarea.value).toContain("Draft");
    expect(screen.getByRole("button", { name: /save draft/i })).toBeTruthy();
  });

  it("disables 'Save draft' until the textarea changes, then enables it", () => {
    const onSaveDraft = vi.fn(noop);
    renderModal({ mode: "edit", draftContent: "initial", onSaveDraft });

    const saveBtn = screen.getByRole("button", { name: /save draft/i }) as HTMLButtonElement;
    expect(saveBtn.disabled).toBe(true);

    const textarea = screen.getByLabelText("Canvas readme markdown editor") as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: "initial + edits" } });

    expect(saveBtn.disabled).toBe(false);
    fireEvent.click(saveBtn);
    expect(onSaveDraft).toHaveBeenCalledWith("initial + edits");
  });

  it("only renders 'Request change' when changeManagementEnabled is true", () => {
    const { unmount } = renderModal({ mode: "edit", changeManagementEnabled: false });
    expect(screen.queryByRole("button", { name: /request change/i })).toBeNull();
    unmount();

    renderModal({ mode: "edit", changeManagementEnabled: true });
    expect(screen.getByRole("button", { name: /request change/i })).toBeTruthy();
  });

  it("asks for confirmation before closing when the draft is dirty", () => {
    const onOpenChange = vi.fn();
    const confirmSpy = vi.fn(() => false);
    window.confirm = confirmSpy;

    renderModal({ mode: "edit", draftContent: "initial", onOpenChange });

    const textarea = screen.getByLabelText("Canvas readme markdown editor") as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: "edits" } });

    fireEvent.click(screen.getByTestId("dialog-close"));

    expect(confirmSpy).toHaveBeenCalledTimes(1);
    expect(onOpenChange).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(screen.getByTestId("dialog-close"));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("closes immediately in live mode without confirmation", () => {
    const onOpenChange = vi.fn();
    const confirmSpy = vi.fn(() => false);
    window.confirm = confirmSpy;

    renderModal({ mode: "live", onOpenChange });

    fireEvent.click(screen.getByTestId("dialog-close"));

    expect(confirmSpy).not.toHaveBeenCalled();
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("forwards the nodes map and linkFor resolver to the markdown renderer", () => {
    renderModal({
      mode: "live",
      nodes: { api: "API", "health-check": "Health Check" },
      linkFor: (slug) => `/demo/canvases/abc?node=${slug}`,
    });

    const body = screen.getByTestId("canvas-markdown");
    expect(body.getAttribute("data-node-slugs")).toBe("api,health-check");
    expect(body.getAttribute("data-node-link-sample")).toBe("/demo/canvases/abc?node=api");
  });
});
