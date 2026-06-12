import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

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

describe("VariablePreview list (list-mode) values", () => {
  it("labels an array value as a list with its item count", () => {
    renderPreview({
      name: "rows",
      value: [
        { name: "a", status: "passed" },
        { name: "b", status: "failed" },
      ],
    });

    expect(screen.getByText("List · 2 items")).toBeTruthy();
  });

  it("offers join+map insert snippets for each field of the first row", () => {
    const onInsertSnippet = vi.fn();
    renderPreview({
      name: "rows",
      value: [{ name: "a", status: "passed" }],
      onInsertSnippet,
    });

    fireEvent.click(screen.getByText("name"));
    expect(onInsertSnippet).toHaveBeenCalledWith('{{ join(rows.map(item, item.name), ", ") }}');
  });

  it("inserts a size() snippet from the Count button", () => {
    const onInsertSnippet = vi.fn();
    renderPreview({ name: "rows", value: [{ name: "a" }], onInsertSnippet });

    fireEvent.click(screen.getByText("Count"));
    expect(onInsertSnippet).toHaveBeenCalledWith("{{ size(rows) }}");
  });

  it("offers a whole-list join for scalar lists", () => {
    const onInsertSnippet = vi.fn();
    renderPreview({ name: "tags", value: ["red", "blue"], onInsertSnippet });

    fireEvent.click(screen.getByText('{{ join(tags, ", ") }}'));
    expect(onInsertSnippet).toHaveBeenCalledWith('{{ join(tags, ", ") }}');
  });

  it("uses bracket access for non-identifier field keys in list snippets", () => {
    const onInsertSnippet = vi.fn();
    renderPreview({
      name: "rows",
      value: [{ "deploy-status": "passed" }],
      onInsertSnippet,
    });

    fireEvent.click(screen.getByText("deploy-status"));
    expect(onInsertSnippet).toHaveBeenCalledWith('{{ join(rows.map(item, item["deploy-status"]), ", ") }}');
  });
});

describe("VariablePreview object values", () => {
  it("uses dot access for identifier-safe field keys", () => {
    const onInsertSnippet = vi.fn();
    renderPreview({ name: "release", value: { status: "passed" }, onInsertSnippet });

    fireEvent.click(screen.getByText("status"));
    expect(onInsertSnippet).toHaveBeenCalledWith("{{ release.status }}");
  });

  it("uses bracket access for non-identifier field keys", () => {
    const onInsertSnippet = vi.fn();
    renderPreview({ name: "release", value: { "deploy-status": "passed", "1up": true }, onInsertSnippet });

    fireEvent.click(screen.getByText("deploy-status"));
    expect(onInsertSnippet).toHaveBeenCalledWith('{{ release["deploy-status"] }}');

    fireEvent.click(screen.getByText("1up"));
    expect(onInsertSnippet).toHaveBeenCalledWith('{{ release["1up"] }}');
  });
});
