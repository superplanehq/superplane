import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ConfigurationField } from "@/api-client";
import { AnyPredicateListFieldRenderer } from "./AnyPredicateListFieldRenderer";

function createField(required: boolean): ConfigurationField {
  return {
    name: "refs",
    label: "Refs",
    type: "any-predicate-list",
    required,
    typeOptions: {
      anyPredicateList: {
        operators: [
          { label: "Equals", value: "equals" },
          { label: "Starts with", value: "starts_with" },
        ],
      },
    },
  };
}

// Row delete buttons are the icon-only buttons; "Add Condition" is the only
// button with text content.
function rowDeleteButtons() {
  return screen.getAllByRole("button").filter((button) => button.textContent === "");
}

describe("AnyPredicateListFieldRenderer", () => {
  it("keeps a required field present as an empty list when the last condition is removed", async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();

    render(
      <AnyPredicateListFieldRenderer
        field={createField(true)}
        value={[{ type: "equals", value: "refs/heads/main" }]}
        onChange={handleChange}
      />,
    );

    await user.click(rowDeleteButtons()[0]);

    // undefined would drop the key from the saved configuration, letting the
    // settings form re-merge the field default as a phantom row.
    expect(handleChange).toHaveBeenCalledWith([]);
  });

  it("shows a hint when a required field has no conditions", () => {
    render(<AnyPredicateListFieldRenderer field={createField(true)} value={[]} onChange={vi.fn()} />);

    expect(screen.getByText("At least one condition is required")).toBeInTheDocument();
  });

  it("does not show the required hint for an optional field with no conditions", () => {
    render(<AnyPredicateListFieldRenderer field={createField(false)} value={[]} onChange={vi.fn()} />);

    expect(screen.queryByText("At least one condition is required")).not.toBeInTheDocument();
  });

  it("clears an optional field with undefined when the last condition is removed", async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();

    render(
      <AnyPredicateListFieldRenderer
        field={createField(false)}
        value={[{ type: "equals", value: "refs/heads/main" }]}
        onChange={handleChange}
      />,
    );

    await user.click(rowDeleteButtons()[0]);

    expect(handleChange).toHaveBeenCalledWith(undefined);
  });

  it("keeps the remaining conditions when one of several is removed", async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();

    render(
      <AnyPredicateListFieldRenderer
        field={createField(true)}
        value={[
          { type: "equals", value: "refs/heads/main" },
          { type: "starts_with", value: "refs/tags/" },
        ]}
        onChange={handleChange}
      />,
    );

    await user.click(rowDeleteButtons()[0]);

    expect(handleChange).toHaveBeenCalledWith([{ type: "starts_with", value: "refs/tags/" }]);
  });
});
