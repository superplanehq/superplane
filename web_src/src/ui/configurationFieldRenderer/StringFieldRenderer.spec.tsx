import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { StringFieldRenderer } from "./StringFieldRenderer";

describe("StringFieldRenderer", () => {
  it("keeps showing default values after switching from read-only to editable", () => {
    const { rerender } = render(
      <StringFieldRenderer
        field={{ name: "title", type: "string", defaultValue: "Untitled" }}
        value={undefined}
        onChange={vi.fn()}
        readOnly
      />,
    );

    expect(screen.getByDisplayValue("Untitled")).toBeInTheDocument();

    rerender(
      <StringFieldRenderer
        field={{ name: "title", type: "string", defaultValue: "Untitled" }}
        value={undefined}
        onChange={vi.fn()}
        readOnly={false}
      />,
    );

    expect(screen.getByDisplayValue("Untitled")).toBeInTheDocument();
  });
});
