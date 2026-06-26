import { act, fireEvent, render, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it, vi } from "vitest";

import type { ConfigurationField } from "@/api-client";

import { ListFieldRenderer } from "./ListFieldRenderer";

function parameterListField(): ConfigurationField {
  return {
    name: "parameters",
    label: "Parameters",
    type: "list",
    typeOptions: {
      list: {
        itemLabel: "Parameter",
        accordion: true,
        reorderable: true,
        itemDefinition: {
          type: "object",
          schema: [
            { name: "name", label: "Name", type: "string", required: true },
            { name: "type", label: "Type", type: "select", required: true },
          ],
        },
      },
    },
  };
}

function stubRowRects() {
  const rows = screen.getAllByTestId("list-item-row");
  rows.forEach((row, index) => {
    row.getBoundingClientRect = vi.fn(
      () =>
        ({
          top: index * 50,
          bottom: index * 50 + 50,
          left: 0,
          right: 200,
          height: 50,
          width: 200,
          x: 0,
          y: index * 50,
          toJSON: () => ({}),
        }) as DOMRect,
    );
  });
  return rows;
}

describe("ListFieldRenderer", () => {
  it("does not warn about switching between controlled and uncontrolled when adding the first item", () => {
    const consoleWarn = vi.spyOn(console, "warn").mockImplementation(() => {});

    function Wrapper() {
      const [value, setValue] = React.useState<unknown[] | undefined>(undefined);
      return (
        <ListFieldRenderer field={parameterListField()} value={value} onChange={(v) => setValue(v as unknown[])} />
      );
    }

    render(<Wrapper />);

    fireEvent.click(screen.getByRole("button", { name: /Add Parameter/i }));

    const controlledWarnings = consoleWarn.mock.calls.filter((call) => {
      const formatted = call.map((arg) => (typeof arg === "string" ? arg : "")).join(" ");
      return formatted.includes("controlled") || formatted.includes("uncontrolled");
    });
    expect(controlledWarnings).toEqual([]);

    consoleWarn.mockRestore();
  });

  it("reorders items immediately as the cursor crosses rows and commits on mouseup", () => {
    const onChange = vi.fn();
    const value = [
      { name: "first", type: "string" },
      { name: "second", type: "string" },
    ];

    render(<ListFieldRenderer field={parameterListField()} value={value} onChange={onChange} />);

    stubRowRects();

    const handle = screen.getByLabelText("Drag to reorder first");
    fireEvent.mouseDown(handle, { button: 0 });

    act(() => {
      window.dispatchEvent(new MouseEvent("mousemove", { clientY: 75 }));
    });

    expect(onChange).not.toHaveBeenCalled();
    const rowsDuringDrag = screen.getAllByTestId("list-item-row");
    expect(rowsDuringDrag[0]).toHaveTextContent("second");
    expect(rowsDuringDrag[1]).toHaveTextContent("first");

    act(() => {
      window.dispatchEvent(new MouseEvent("mouseup"));
    });

    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenCalledWith([
      { name: "second", type: "string" },
      { name: "first", type: "string" },
    ]);
  });

  it("does not show drag handles when reorderable is false", () => {
    const field = parameterListField();
    field.typeOptions!.list!.reorderable = false;

    render(
      <ListFieldRenderer
        field={field}
        value={[
          { name: "first", type: "string" },
          { name: "second", type: "string" },
        ]}
        onChange={vi.fn()}
      />,
    );

    expect(screen.queryByLabelText(/Drag to reorder/)).not.toBeInTheDocument();
  });
});
