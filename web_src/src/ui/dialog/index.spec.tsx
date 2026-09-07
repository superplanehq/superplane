import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { MockInstance } from "vitest";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/ui/dialog";

describe("ui dialog accessibility", () => {
  let warnSpy: MockInstance<typeof console.warn>;

  beforeEach(() => {
    warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
  });

  afterEach(() => {
    warnSpy.mockRestore();
  });

  it("adds a hidden fallback title when a dialog title is missing", () => {
    render(
      <Dialog open>
        <DialogContent>
          <div>Body</div>
        </DialogContent>
      </Dialog>,
    );

    expect(screen.getByText("Dialog")).toBeInTheDocument();
    expect(screen.getByText("Body")).toBeInTheDocument();
  });

  it("does not add the fallback title when a dialog title is already present", () => {
    render(
      <Dialog open>
        <DialogContent>
          <DialogTitle>Explicit title</DialogTitle>
          <div>Body</div>
        </DialogContent>
      </Dialog>,
    );

    expect(screen.getByText("Explicit title")).toBeInTheDocument();
    expect(screen.queryByText("Dialog")).not.toBeInTheDocument();
  });

  it("does not emit a missing description warning when no DialogDescription is provided", () => {
    render(
      <Dialog open>
        <DialogContent>
          <DialogTitle>Title</DialogTitle>
          <div>Body</div>
        </DialogContent>
      </Dialog>,
    );

    const describedByWarning = warnSpy.mock.calls
      .flat()
      .some((entry) => typeof entry === "string" && entry.includes("aria-describedby"));
    expect(describedByWarning).toBe(false);
  });

  it("links aria-describedby to the provided DialogDescription", () => {
    render(
      <Dialog open>
        <DialogContent>
          <DialogTitle>Title</DialogTitle>
          <DialogDescription>Helpful description</DialogDescription>
          <div>Body</div>
        </DialogContent>
      </Dialog>,
    );

    const dialog = screen.getByRole("dialog");
    const describedBy = dialog.getAttribute("aria-describedby");
    expect(describedBy).toBeTruthy();
    const description = document.getElementById(describedBy ?? "");
    expect(description?.textContent).toBe("Helpful description");
  });

  it("uses theme surface tokens instead of a hardcoded dark gray fill", () => {
    render(
      <Dialog open>
        <DialogContent>
          <DialogTitle>Title</DialogTitle>
          <div>Body</div>
        </DialogContent>
      </Dialog>,
    );

    const dialog = screen.getByRole("dialog");
    expect(dialog.className).toContain("bg-popover");
    expect(dialog.className).toContain("border-border");
    expect(dialog.className).not.toContain("dark:bg-gray-900");
  });
});
