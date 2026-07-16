import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SelectFieldRenderer } from "./SelectFieldRenderer";

describe("SelectFieldRenderer", () => {
  it("does not apply defaults after switching from read-only to editable", () => {
    const handleChange = vi.fn();
    const { rerender } = render(
      <SelectFieldRenderer
        field={{
          name: "mode",
          type: "select",
          defaultValue: "auto",
          typeOptions: { select: { options: [{ label: "Auto", value: "auto" }] } },
        }}
        value={undefined}
        onChange={handleChange}
        readOnly
      />,
    );

    rerender(
      <SelectFieldRenderer
        field={{
          name: "mode",
          type: "select",
          defaultValue: "auto",
          typeOptions: { select: { options: [{ label: "Auto", value: "auto" }] } },
        }}
        value={undefined}
        onChange={handleChange}
        readOnly={false}
      />,
    );

    expect(handleChange).not.toHaveBeenCalled();
  });
});
