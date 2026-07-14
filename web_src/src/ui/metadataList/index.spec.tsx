import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { MetadataList, type MetadataItem } from "./index";

describe("MetadataList", () => {
  it("renders string labels", () => {
    render(<MetadataList items={[{ icon: "sparkles", label: "claude-opus-4-6" }]} />);
    expect(screen.getByText("claude-opus-4-6")).toBeInTheDocument();
  });

  it("does not crash when a label is an unresolved integration-resource object", () => {
    // Regression for issue #6072: rendering a raw { id, name, type } object as a
    // React child throws "Objects are not valid as a React child" and crashes
    // the whole canvas. The guard must coerce it to the resource name instead.
    const badItem = {
      icon: "sparkles",
      label: { id: "abc123", name: "claude-opus-4-6", type: "model" },
    } as unknown as MetadataItem;

    expect(() => render(<MetadataList items={[badItem]} />)).not.toThrow();
    expect(screen.getByText("claude-opus-4-6")).toBeInTheDocument();
  });

  it("still renders React node labels", () => {
    render(<MetadataList items={[{ icon: "sparkles", label: <span>custom node</span> }]} />);
    expect(screen.getByText("custom node")).toBeInTheDocument();
  });
});
