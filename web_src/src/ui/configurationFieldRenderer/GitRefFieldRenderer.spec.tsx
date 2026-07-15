import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import type { ConfigurationField } from "@/api-client";

import { GitRefFieldRenderer } from "./GitRefFieldRenderer";

const EXPRESSION_VALUE = '{{ $["node"].data.ref }}';

function gitRefField(): ConfigurationField {
  return {
    name: "deployRef",
    label: "Deploy ref",
    type: "git-ref",
  };
}

function ControlledRenderer({ initialValue }: { initialValue?: string }) {
  const [value, setValue] = useState<string | undefined>(initialValue);
  return (
    <>
      <span data-testid="current-value">{value ?? ""}</span>
      <GitRefFieldRenderer
        field={gitRefField()}
        value={value}
        onChange={(nextValue) => setValue(nextValue as string | undefined)}
        allowExpressions
      />
    </>
  );
}

describe("GitRefFieldRenderer", () => {
  it("renders fixed branch/tag picker when expressions are disabled", () => {
    render(<GitRefFieldRenderer field={gitRefField()} value="refs/heads/main" onChange={vi.fn()} />);

    expect(screen.getByRole("combobox")).toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "Expression" })).not.toBeInTheDocument();
  });

  it("starts in expression mode when the value is an expression", () => {
    render(<ControlledRenderer initialValue={EXPRESSION_VALUE} />);

    expect(screen.getByRole("textbox")).toHaveValue(EXPRESSION_VALUE);
    expect(screen.getByRole("tab", { name: "Expression" })).toHaveAttribute("aria-selected", "true");
  });

  it("preserves a fixed ref when toggling to Expression and back", async () => {
    const user = userEvent.setup();
    render(<ControlledRenderer initialValue="refs/heads/main" />);

    expect(screen.getByTestId("current-value").textContent).toBe("refs/heads/main");

    await user.click(screen.getByRole("tab", { name: "Expression" }));

    expect(screen.getByTestId("current-value").textContent).toBe("refs/heads/main");
    expect(screen.getByRole("textbox")).toHaveValue("refs/heads/main");

    await user.click(screen.getByRole("tab", { name: "Fixed" }));

    expect(screen.getByTestId("current-value").textContent).toBe("refs/heads/main");
  });

  it("preserves an expression value when toggling to Fixed and back", async () => {
    const user = userEvent.setup();
    render(<ControlledRenderer initialValue={EXPRESSION_VALUE} />);

    expect(screen.getByRole("textbox")).toHaveValue(EXPRESSION_VALUE);

    await user.click(screen.getByRole("tab", { name: "Fixed" }));

    expect(screen.getByTestId("current-value").textContent).toBe(EXPRESSION_VALUE);

    await user.click(screen.getByRole("tab", { name: "Expression" }));

    expect(screen.getByRole("textbox")).toHaveValue(EXPRESSION_VALUE);
  });

  it("never clears the value via onChange when switching modes", async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();
    render(
      <GitRefFieldRenderer field={gitRefField()} value="refs/heads/main" onChange={handleChange} allowExpressions />,
    );

    await user.click(screen.getByRole("tab", { name: "Expression" }));
    await user.click(screen.getByRole("tab", { name: "Fixed" }));

    expect(handleChange).not.toHaveBeenCalledWith(undefined);
  });
});
