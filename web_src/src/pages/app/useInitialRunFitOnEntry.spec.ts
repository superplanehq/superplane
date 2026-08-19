import { renderHook } from "@testing-library/react";
import { useRef } from "react";
import { describe, expect, it, vi } from "vitest";
import { useInitialRunFitOnEntry } from "./useInitialRunFitOnEntry";

const runId = "550e8400-e29b-41d4-a716-446655440000";

function renderWithRef(
  initialProps: {
    isRunInspectionMode: boolean;
    selectedRunId: string | null;
    searchParams: URLSearchParams;
  },
  requestRunFit: (runId: string) => void,
) {
  return renderHook(
    (props: { isRunInspectionMode: boolean; selectedRunId: string | null; searchParams: URLSearchParams }) => {
      const requestRunFitRef = useRef(requestRunFit);
      requestRunFitRef.current = requestRunFit;
      useInitialRunFitOnEntry({ ...props, requestRunFitRef });
    },
    { initialProps },
  );
}

describe("useInitialRunFitOnEntry", () => {
  it("requests a fit once on mount when landing directly on a run URL", () => {
    const requestRunFit = vi.fn();

    renderWithRef(
      {
        isRunInspectionMode: true,
        selectedRunId: runId,
        searchParams: new URLSearchParams({ run: runId }),
      },
      requestRunFit,
    );

    expect(requestRunFit).toHaveBeenCalledTimes(1);
    expect(requestRunFit).toHaveBeenCalledWith(runId);
  });

  it("does not request a fit when not in run inspection mode", () => {
    const requestRunFit = vi.fn();

    renderWithRef(
      {
        isRunInspectionMode: false,
        selectedRunId: null,
        searchParams: new URLSearchParams(),
      },
      requestRunFit,
    );

    expect(requestRunFit).not.toHaveBeenCalled();
  });

  it("does not request a fit when a node is already pending focus via the sidebar", () => {
    const requestRunFit = vi.fn();

    renderWithRef(
      {
        isRunInspectionMode: true,
        selectedRunId: runId,
        searchParams: new URLSearchParams({ run: runId, sidebar: "1", node: "node-a" }),
      },
      requestRunFit,
    );

    expect(requestRunFit).not.toHaveBeenCalled();
  });

  it("does not re-fire on rerender, even if the run id changes", () => {
    const requestRunFit = vi.fn();

    const { rerender } = renderWithRef(
      {
        isRunInspectionMode: true,
        selectedRunId: runId,
        searchParams: new URLSearchParams({ run: runId }),
      },
      requestRunFit,
    );

    expect(requestRunFit).toHaveBeenCalledTimes(1);

    const otherRunId = "660e8400-e29b-41d4-a716-446655440111";
    rerender({
      isRunInspectionMode: true,
      selectedRunId: otherRunId,
      searchParams: new URLSearchParams({ run: otherRunId }),
    });

    expect(requestRunFit).toHaveBeenCalledTimes(1);
  });

  it("does not fire when mounted without a selected run", () => {
    const requestRunFit = vi.fn();

    renderWithRef(
      {
        isRunInspectionMode: true,
        selectedRunId: null,
        searchParams: new URLSearchParams(),
      },
      requestRunFit,
    );

    expect(requestRunFit).not.toHaveBeenCalled();
  });
});
