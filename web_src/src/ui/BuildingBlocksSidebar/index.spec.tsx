import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { BuildingBlocksSidebar } from "./index";
import type { BuildingBlockCategory } from "./types";

const defaultProps = {
  isOpen: true,
  onToggle: vi.fn(),
  blocks: [] as BuildingBlockCategory[],
  canvasZoom: 1,
};

const coreCategory: BuildingBlockCategory = {
  name: "Core",
  blocks: [
    { name: "manual", label: "Manual Run", type: "trigger" },
    { name: "filter", label: "Filter", type: "component" },
    { name: "approval", label: "Approval", type: "component" },
  ],
};

async function openSidebarSettings(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByLabelText("Sidebar settings"));
}

function getSidebarSetting(name: string) {
  return screen.getByRole("menuitemcheckbox", { name });
}

describe("BuildingBlocksSidebar", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("calls onToggle(false) when close button is clicked while disabled", () => {
    const onToggle = vi.fn();
    render(
      <BuildingBlocksSidebar
        {...defaultProps}
        onToggle={onToggle}
        disabled={true}
        disabledMessage="You don't have permission to edit this canvas."
      />,
    );

    const closeButton = screen.getByTestId("close-sidebar-button");
    fireEvent.click(closeButton);

    expect(onToggle).toHaveBeenCalledWith(false);
  });

  it("calls onToggle(false) when close button is clicked while not disabled", () => {
    const onToggle = vi.fn();
    render(<BuildingBlocksSidebar {...defaultProps} onToggle={onToggle} disabled={false} />);

    const closeButton = screen.getByTestId("close-sidebar-button");
    fireEvent.click(closeButton);

    expect(onToggle).toHaveBeenCalledWith(false);
  });

  it("renders the disabled overlay when disabled", () => {
    const { container } = render(
      <BuildingBlocksSidebar
        {...defaultProps}
        disabled={true}
        disabledMessage="You don't have permission to edit this canvas."
      />,
    );

    expect(container.querySelector(".cursor-not-allowed")).toBeInTheDocument();
  });

  it("does not render when isOpen is false", () => {
    const { container } = render(<BuildingBlocksSidebar {...defaultProps} isOpen={false} />);

    expect(container.querySelector('[data-testid="building-blocks-sidebar"]')).not.toBeInTheDocument();
  });

  it("persists display settings after remounting", async () => {
    const user = userEvent.setup();
    const { unmount } = render(<BuildingBlocksSidebar {...defaultProps} blocks={[coreCategory]} />);

    await openSidebarSettings(user);
    await user.click(getSidebarSetting("Show integration setup status"));
    await openSidebarSettings(user);
    await user.click(getSidebarSetting("Connected integrations on top"));

    unmount();

    render(<BuildingBlocksSidebar {...defaultProps} blocks={[coreCategory]} />);

    await openSidebarSettings(user);
    expect(getSidebarSetting("Show integration setup status")).toHaveAttribute("aria-checked", "false");
    expect(getSidebarSetting("Connected integrations on top")).toHaveAttribute("aria-checked", "true");
  });

  describe("Enter-to-submit", () => {
    it("calls onEnterSubmit with the first visible block when Enter is pressed after typing", () => {
      const onEnterSubmit = vi.fn();
      render(<BuildingBlocksSidebar {...defaultProps} blocks={[coreCategory]} onEnterSubmit={onEnterSubmit} />);

      const input = screen.getByPlaceholderText("Filter components...");
      fireEvent.change(input, { target: { value: "filt" } });
      fireEvent.keyDown(input, { key: "Enter" });

      expect(onEnterSubmit).toHaveBeenCalledTimes(1);
      expect(onEnterSubmit.mock.calls[0][0]).toMatchObject({ name: "filter" });
    });

    it("is a no-op when the filter is empty", () => {
      const onEnterSubmit = vi.fn();
      render(<BuildingBlocksSidebar {...defaultProps} blocks={[coreCategory]} onEnterSubmit={onEnterSubmit} />);

      const input = screen.getByPlaceholderText("Filter components...");
      fireEvent.keyDown(input, { key: "Enter" });

      expect(onEnterSubmit).not.toHaveBeenCalled();
    });

    it("is a no-op when the filter matches nothing", () => {
      const onEnterSubmit = vi.fn();
      render(<BuildingBlocksSidebar {...defaultProps} blocks={[coreCategory]} onEnterSubmit={onEnterSubmit} />);

      const input = screen.getByPlaceholderText("Filter components...");
      fireEvent.change(input, { target: { value: "zzzzzz" } });
      fireEvent.keyDown(input, { key: "Enter" });

      expect(onEnterSubmit).not.toHaveBeenCalled();
    });

    it("is a no-op when the sidebar is disabled", () => {
      const onEnterSubmit = vi.fn();
      render(
        <BuildingBlocksSidebar
          {...defaultProps}
          blocks={[coreCategory]}
          disabled={true}
          onEnterSubmit={onEnterSubmit}
        />,
      );

      const input = screen.getByPlaceholderText("Filter components...");
      fireEvent.change(input, { target: { value: "filt" } });
      fireEvent.keyDown(input, { key: "Enter" });

      expect(onEnterSubmit).not.toHaveBeenCalled();
    });
  });
});
