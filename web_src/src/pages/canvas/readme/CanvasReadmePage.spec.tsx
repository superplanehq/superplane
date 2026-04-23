import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { canActMock, versionParamRef, draftReadmeRef } = vi.hoisted(() => ({
  canActMock: vi.fn<(resource: string, action: string) => boolean>(),
  versionParamRef: { current: "" as string },
  draftReadmeRef: {
    current: {
      data: undefined as
        | { canvasId: string; versionId: string; versionState: string; content: string }
        | undefined,
      isLoading: false,
    },
  },
}));

vi.mock("react-router-dom", () => ({
  Link: ({ children, to, onClick }: { children: ReactNode; to: string; onClick?: () => void }) => (
    <a href={to} onClick={onClick}>
      {children}
    </a>
  ),
  useNavigate: () => vi.fn(),
  useParams: () => ({ organizationId: "org-123", canvasId: "canvas-abc" }),
  useSearchParams: () => [new URLSearchParams(versionParamRef.current ? `version=${versionParamRef.current}` : "")],
}));

vi.mock("@/hooks/usePageTitle", () => ({
  usePageTitle: () => undefined,
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganization: () => ({
    data: { metadata: { id: "org-123", name: "Acme Corp" } },
    isLoading: false,
  }),
}));

vi.mock("@/contexts/PermissionsContext", () => ({
  usePermissions: () => ({ canAct: canActMock, isLoading: false }),
}));

vi.mock("../settings/PageHeader", () => ({
  PageHeader: ({ title }: { title: string }) => <header>{title}</header>,
}));

//
// Render the markdown as a plain <pre> so the test can assert on the
// final body content without having to depend on the real markdown pipeline.
//
vi.mock("@/ui/Markdown/CanvasMarkdown", () => ({
  CanvasMarkdown: ({ children }: { children: string }) => (
    <pre data-testid="canvas-markdown">{children}</pre>
  ),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvas: () => ({
    data: {
      metadata: { id: "canvas-abc", name: "Health Check Monitor" },
      spec: { nodes: [], edges: [], changeManagement: { enabled: false } },
    },
    isLoading: false,
    error: null,
  }),
  useCanvasReadme: (_canvasId: string, versionOrDraft: string) => {
    if (versionOrDraft === "draft") {
      return draftReadmeRef.current;
    }
    return {
      data: {
        canvasId: "canvas-abc",
        versionId: "live-v1",
        versionState: "published",
        content: "# Live readme\nPublished content.",
      },
      isLoading: false,
    };
  },
  useUpdateCanvasReadme: () => ({
    mutateAsync: vi.fn().mockResolvedValue(undefined),
    isPending: false,
  }),
  useCreateCanvasChangeRequest: () => ({
    mutateAsync: vi.fn().mockResolvedValue(undefined),
    isPending: false,
  }),
}));

import { CanvasReadmePage } from "./index";

beforeEach(() => {
  canActMock.mockReset();
  versionParamRef.current = "";
  draftReadmeRef.current = { data: undefined, isLoading: false };
});

describe("CanvasReadmePage mode selection", () => {
  it("renders the read-only live view when there is no ?version param", () => {
    canActMock.mockReturnValue(true);
    draftReadmeRef.current = {
      data: {
        canvasId: "canvas-abc",
        versionId: "draft-xyz",
        versionState: "draft",
        content: "# Draft readme\nWork in progress.",
      },
      isLoading: false,
    };

    render(<CanvasReadmePage />);

    expect(screen.queryByLabelText("Canvas readme markdown editor")).toBeNull();
    expect(screen.queryByRole("button", { name: /save draft/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /view live/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /edit draft/i })).toBeNull();

    const markdown = screen.getByTestId("canvas-markdown");
    expect(markdown.textContent).toContain("Live readme");
    expect(markdown.textContent).not.toContain("Draft readme");

    const backLink = screen.getByRole("link", { name: /back to canvas/i });
    expect(backLink.getAttribute("href")).toBe("/org-123/canvases/canvas-abc");
  });

  it("renders the editor when ?version matches the caller's draft and update is allowed", () => {
    canActMock.mockReturnValue(true);
    versionParamRef.current = "draft-xyz";
    draftReadmeRef.current = {
      data: {
        canvasId: "canvas-abc",
        versionId: "draft-xyz",
        versionState: "draft",
        content: "# Draft readme\nWork in progress.",
      },
      isLoading: false,
    };

    render(<CanvasReadmePage />);

    const textarea = screen.getByLabelText("Canvas readme markdown editor") as HTMLTextAreaElement;
    expect(textarea).toBeTruthy();
    expect(textarea.value).toContain("Draft readme");

    expect(screen.getByRole("button", { name: /save draft/i })).toBeTruthy();

    // No mode toggle, no Publish, no View live.
    expect(screen.queryByRole("button", { name: /view live/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /edit draft/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /^publish/i })).toBeNull();

    // Change management is disabled in the mocked canvas, so no "Request change" button either.
    expect(screen.queryByRole("button", { name: /request change/i })).toBeNull();

    const backLink = screen.getByRole("link", { name: /back to canvas/i });
    expect(backLink.getAttribute("href")).toBe("/org-123/canvases/canvas-abc?version=draft-xyz");
  });

  it("collapses to the live view when ?version is present but the user lacks update permission", () => {
    canActMock.mockImplementation((resource, action) => !(resource === "canvases" && action === "update"));
    versionParamRef.current = "draft-xyz";
    draftReadmeRef.current = { data: undefined, isLoading: false };

    render(<CanvasReadmePage />);

    expect(screen.queryByLabelText("Canvas readme markdown editor")).toBeNull();
    expect(screen.queryByRole("button", { name: /save draft/i })).toBeNull();

    const markdown = screen.getByTestId("canvas-markdown");
    expect(markdown.textContent).toContain("Live readme");
  });

  it("collapses to the live view when ?version points at a different draft than the caller's", () => {
    canActMock.mockReturnValue(true);
    versionParamRef.current = "someone-elses-draft";
    draftReadmeRef.current = {
      data: {
        canvasId: "canvas-abc",
        versionId: "draft-xyz",
        versionState: "draft",
        content: "# Draft readme\nWork in progress.",
      },
      isLoading: false,
    };

    render(<CanvasReadmePage />);

    expect(screen.queryByLabelText("Canvas readme markdown editor")).toBeNull();
    expect(screen.queryByRole("button", { name: /save draft/i })).toBeNull();
  });
});
