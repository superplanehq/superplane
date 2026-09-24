import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { clearWorkOrderFileDownloadCache } from "@/lib/workOrderFiles";
import { clearWorkspaceMarkdownImageLoadCache } from "@/lib/workspaceMarkdownImages";

import { WorkOrderDescription } from "./WorkOrderDescription";

let notifyResize: () => void;

class MockResizeObserver {
  constructor(callback: ResizeObserverCallback) {
    notifyResize = () => callback([], this as unknown as ResizeObserver);
  }

  observe() {}
  unobserve() {}
  disconnect() {}
}

describe("WorkOrderDescription", () => {
  beforeEach(() => {
    vi.stubGlobal("ResizeObserver", MockResizeObserver);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    clearWorkOrderFileDownloadCache();
    clearWorkspaceMarkdownImageLoadCache();
  });

  it("updates the collapse control when the rendered content height changes", () => {
    let contentHeight = 300;
    render(<WorkOrderDescription description="# Test description" />);

    const content = screen.getByTestId("work-order-description-markdown").parentElement;
    expect(content).not.toBeNull();
    Object.defineProperty(content!, "scrollHeight", {
      configurable: true,
      get: () => contentHeight,
    });

    act(() => notifyResize());
    expect(screen.getByRole("button", { name: /show more/i })).toBeInTheDocument();

    contentHeight = 100;
    act(() => notifyResize());
    expect(screen.queryByRole("button", { name: /show more/i })).not.toBeInTheDocument();
  });

  it("does not add Show more when collapse is off", () => {
    render(
      <div data-testid="description-pane" style={{ overflowY: "auto", padding: "24px 0" }}>
        <WorkOrderDescription description="# Test description" collapsible={false} />
      </div>,
    );

    const pane = screen.getByTestId("description-pane");
    Object.defineProperty(pane, "clientHeight", { configurable: true, get: () => 520 });
    const content = screen.getByTestId("work-order-description-markdown").parentElement;
    Object.defineProperty(content!, "scrollHeight", { configurable: true, get: () => 640 });

    act(() => notifyResize());
    expect(screen.queryByRole("button", { name: /show more/i })).not.toBeInTheDocument();
    expect(content).not.toHaveStyle({ maxHeight: "440px" });
  });

  it("keeps the body open when it fits the scroll pane", () => {
    const contentHeight = 360;
    render(
      <div data-testid="description-pane" style={{ overflowY: "auto", padding: "24px 0" }}>
        <WorkOrderDescription description="# Test description" />
      </div>,
    );

    const pane = screen.getByTestId("description-pane");
    Object.defineProperty(pane, "clientHeight", { configurable: true, get: () => 520 });
    const content = screen.getByTestId("work-order-description-markdown").parentElement;
    Object.defineProperty(content!, "scrollHeight", { configurable: true, get: () => contentHeight });

    act(() => notifyResize());
    expect(screen.queryByRole("button", { name: /show more/i })).not.toBeInTheDocument();
    expect(content).not.toHaveStyle({ maxHeight: "220px" });
  });

  it("fills the leftover pane before Show more when the body is taller", () => {
    render(
      <div data-testid="description-pane" style={{ overflowY: "auto", padding: "24px 0" }}>
        <WorkOrderDescription description="# Test description" />
      </div>,
    );

    const pane = screen.getByTestId("description-pane");
    Object.defineProperty(pane, "clientHeight", { configurable: true, get: () => 520 });
    const content = screen.getByTestId("work-order-description-markdown").parentElement;
    Object.defineProperty(content!, "scrollHeight", { configurable: true, get: () => 640 });

    act(() => notifyResize());
    expect(screen.getByRole("button", { name: /show more/i })).toBeInTheDocument();
    expect(content).toHaveStyle({ maxHeight: "440px" });
  });

  it("clamps to a preview height when one is set", () => {
    render(
      <div data-testid="description-pane" style={{ overflowY: "auto", padding: "24px 0" }}>
        <WorkOrderDescription description="# Test description" previewHeight={220} />
      </div>,
    );

    const pane = screen.getByTestId("description-pane");
    Object.defineProperty(pane, "clientHeight", { configurable: true, get: () => 520 });
    const content = screen.getByTestId("work-order-description-markdown").parentElement;
    Object.defineProperty(content!, "scrollHeight", { configurable: true, get: () => 640 });

    act(() => notifyResize());
    expect(screen.getByRole("button", { name: /show more/i })).toBeInTheDocument();
    expect(content).toHaveStyle({ maxHeight: "220px" });
  });

  it("uses the outgoing fade class on a collapsed preview", () => {
    render(
      <WorkOrderDescription
        description="# Test description"
        previewHeight={220}
        fadeClassName="sp-chat-outgoing-fade"
      />,
    );

    const content = screen.getByTestId("work-order-description-markdown").parentElement;
    Object.defineProperty(content!, "scrollHeight", { configurable: true, get: () => 640 });

    act(() => notifyResize());
    expect(screen.getByTestId("work-order-description").querySelector(".sp-chat-outgoing-fade")).not.toBeNull();
  });

  it("does not collapse a preview when the body fits", () => {
    render(<WorkOrderDescription description="# Test description" previewHeight={220} />);

    const content = screen.getByTestId("work-order-description-markdown").parentElement;
    Object.defineProperty(content!, "scrollHeight", { configurable: true, get: () => 180 });

    act(() => notifyResize());
    expect(screen.queryByRole("button", { name: /show more/i })).not.toBeInTheDocument();
    expect(content).not.toHaveStyle({ maxHeight: "220px" });
  });

  it("leaves room for checks in the leftover pane", () => {
    render(
      <div data-testid="description-pane" style={{ overflowY: "auto", padding: "24px 0" }}>
        <WorkOrderDescription description="# Test description" />
        <section data-testid="checks" style={{ marginTop: 40 }}>
          Checks
        </section>
      </div>,
    );

    const pane = screen.getByTestId("description-pane");
    Object.defineProperty(pane, "clientHeight", { configurable: true, get: () => 520 });
    Object.defineProperty(screen.getByTestId("checks"), "offsetHeight", { configurable: true, get: () => 120 });
    const content = screen.getByTestId("work-order-description-markdown").parentElement;
    Object.defineProperty(content!, "scrollHeight", { configurable: true, get: () => 400 });

    act(() => notifyResize());
    expect(screen.getByRole("button", { name: /show more/i })).toBeInTheDocument();
    expect(content).toHaveStyle({ maxHeight: "280px" });
  });

  it("uses the download URL as the image source when files are present", () => {
    const fileId = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa";
    render(
      <WorkOrderDescription
        description={`See ![bug](sp-file://${fileId})`}
        files={[{ id: fileId, downloadUrl: "https://cdn.example/bug.png" }]}
        collapsible={false}
      />,
    );

    expect(screen.getByRole("img", { name: "bug" })).toHaveAttribute("src", "https://cdn.example/bug.png");
  });

  it("does not render an empty image source without files", () => {
    render(
      <WorkOrderDescription
        description="See ![bug](sp-file://aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa)"
        collapsible={false}
      />,
    );

    expect(screen.queryByRole("img")).not.toBeInTheDocument();
    expect(screen.getByText("bug")).toBeInTheDocument();
  });

  it("keeps the image source when a file URL is reminted", () => {
    const id = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa";
    const description = `![Screenshot](sp-file://${id})`;
    const first = "https://files.example/screenshot.png?expires=9999999999&sig=one";
    const reminted = "https://files.example/screenshot.png?expires=9999999999&sig=two";
    const { rerender } = render(
      <WorkOrderDescription description={description} files={[{ id, downloadUrl: first }]} collapsible={false} />,
    );
    const image = screen.getByRole("img", { name: "Screenshot" });

    expect(image).toHaveAttribute("src", first);

    rerender(
      <WorkOrderDescription description={description} files={[{ id, downloadUrl: reminted }]} collapsible={false} />,
    );

    expect(image).toHaveAttribute("src", first);
  });

  it("keeps a loaded request image visible when the chat wrapper remounts", () => {
    const id = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa";
    const description = `![Screenshot](sp-file://${id})`;
    const src = "https://files.example/screenshot.png?expires=9999999999&sig=one";
    const { rerender } = render(
      <WorkOrderDescription key="poll-1" description={description} files={[{ id, downloadUrl: src }]} />,
    );

    fireEvent.load(screen.getByRole("img", { name: "Screenshot" }));
    rerender(<WorkOrderDescription key="poll-2" description={description} files={[{ id, downloadUrl: src }]} />);

    expect(screen.getByRole("img", { name: "Screenshot" })).not.toHaveClass("opacity-0");
  });
});
