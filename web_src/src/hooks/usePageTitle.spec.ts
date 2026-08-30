import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { usePageTitle } from "@/hooks/usePageTitle";

describe("usePageTitle", () => {
  it("joins parts with middots and appends SuperPlane", () => {
    renderHook(() => usePageTitle(["Task #12", "Acme"]));
    expect(document.title).toBe("Task #12 · Acme · SuperPlane");
  });

  it("drops empty/undefined/null parts", () => {
    renderHook(() => usePageTitle([undefined, "  ", "Overview", null]));
    expect(document.title).toBe("Overview · SuperPlane");
  });

  it("falls back to just SuperPlane when no parts are provided", () => {
    renderHook(() => usePageTitle([]));
    expect(document.title).toBe("SuperPlane");
  });

  it("re-fires when the derived title string changes across renders", () => {
    const { rerender } = renderHook(({ parts }: { parts: Array<string | undefined> }) => usePageTitle(parts), {
      initialProps: { parts: ["Tasks", "Acme"] },
    });
    expect(document.title).toBe("Tasks · Acme · SuperPlane");

    rerender({ parts: ["Lines", "Acme"] });
    expect(document.title).toBe("Lines · Acme · SuperPlane");
  });

  it("does not write document.title when enabled is false", () => {
    document.title = "Existing Title";
    renderHook(() => usePageTitle(["Ignored"], { enabled: false }));
    expect(document.title).toBe("Existing Title");
  });

  it("resumes writing document.title once enabled flips back to true", () => {
    const { rerender } = renderHook(
      ({ enabled }: { enabled: boolean }) => usePageTitle(["Settings", "Acme"], { enabled }),
      { initialProps: { enabled: false } },
    );
    document.title = "Set by a child page";
    expect(document.title).toBe("Set by a child page");

    rerender({ enabled: true });
    expect(document.title).toBe("Settings · Acme · SuperPlane");
  });
});
