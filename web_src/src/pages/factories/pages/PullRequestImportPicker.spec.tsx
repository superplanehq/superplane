import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import { PullRequestImportResults, type PullRequestImportResultsProps } from "./PullRequestImportPicker";

const items = [
  { url: "https://github.com/acme/app/pull/42", title: "Fix retries", number: "42", draft: true },
  { url: "https://github.com/acme/app/pull/43", title: "Fix imports", number: "43", existingTask: { id: "task-43" } },
];

function results(overrides: Partial<PullRequestImportResultsProps> = {}) {
  return (
    <PullRequestImportResults
      items={items}
      query=""
      onQueryChange={vi.fn()}
      onLoadMore={vi.fn()}
      onRetry={vi.fn()}
      onSelect={vi.fn()}
      {...overrides}
    />
  );
}

describe("Pull request import results", () => {
  it("allows selection of drafts and existing tasks", () => {
    const onSelect = vi.fn();
    render(results({ onSelect }));
    fireEvent.click(screen.getByRole("button", { name: /Fix retries/ }));
    expect(onSelect).toHaveBeenLastCalledWith(items[0]);
    fireEvent.click(screen.getByRole("button", { name: /Already imported/ }));
    expect(onSelect).toHaveBeenLastCalledWith(items[1]);
  });

  it("blocks repeated selection while an import is pending", () => {
    const onSelect = vi.fn();
    render(results({ onSelect, isPending: true }));
    fireEvent.click(screen.getByRole("button", { name: /Fix retries/ }));
    fireEvent.click(screen.getByRole("button", { name: /Fix imports/ }));
    expect(onSelect).not.toHaveBeenCalled();
    expect(screen.getByRole("textbox", { name: "Search pull requests" })).toBeDisabled();
    expect(screen.getByRole("status")).toBeVisible();
  });

  it("offers retry after connection failure", () => {
    const onRetry = vi.fn();
    render(results({ items: [], error: "Connect GitHub and try again.", onRetry }));
    expect(screen.getByRole("alert")).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("passes title, number, and URL input to search", () => {
    const onQueryChange = vi.fn();
    render(results({ onQueryChange }));
    const input = screen.getByRole("textbox", { name: "Search pull requests" });
    for (const query of ["retry", "#42", items[0].url]) {
      fireEvent.change(input, { target: { value: query } });
      expect(onQueryChange).toHaveBeenLastCalledWith(query);
    }
  });
});
