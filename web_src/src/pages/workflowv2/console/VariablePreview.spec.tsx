import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { VariablePreview } from "./VariablePreview";

function renderPreview(props: Partial<React.ComponentProps<typeof VariablePreview>> = {}) {
  return render(
    <VariablePreview name="release" value={undefined} loading={false} onInsertSnippet={() => {}} {...props} />,
  );
}

describe("VariablePreview loading state", () => {
  it("shows the loading message while a variable resolves to null in flight", () => {
    // `useMarkdownVariables` resolves in-flight variables to `null` (not
    // `undefined`), so a fetch with `loading` true and `value` null must still
    // surface "Loading preview…" rather than "No data resolved yet.".
    renderPreview({ value: null, loading: true });

    expect(screen.getByText("Loading preview…")).toBeTruthy();
    expect(screen.queryByText("No data resolved yet.")).toBeNull();
  });

  it("shows the loading message while a variable is undefined in flight", () => {
    renderPreview({ value: undefined, loading: true });

    expect(screen.getByText("Loading preview…")).toBeTruthy();
  });

  it("shows the empty message when loading settles with no data", () => {
    renderPreview({ value: null, loading: false });

    expect(screen.getByText("No data resolved yet.")).toBeTruthy();
    expect(screen.queryByText("Loading preview…")).toBeNull();
  });

  it("keeps rendering resolved fields during a background refetch", () => {
    // A non-null value during a refetch (loading true) must not flash the
    // loading text — render the already-resolved fields instead.
    renderPreview({ value: { service: "api" }, loading: true });

    expect(screen.queryByText("Loading preview…")).toBeNull();
    expect(screen.getByText("service")).toBeTruthy();
    expect(screen.getByText("api")).toBeTruthy();
  });

  it("prefers the error message over the empty message once settled", () => {
    renderPreview({ value: null, loading: false, error: "No runs found yet." });

    expect(screen.getByTestId("markdown-variable-preview-error").textContent).toBe("No runs found yet.");
  });
});
