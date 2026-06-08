import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { MultiCombobox } from "./multi-combobox";

const options = [
  { id: "1", name: "Apple" },
  { id: "2", name: "Banana" },
  { id: "3", name: "Lemon" },
];

const displayValue = (item: { name: string }) => item.name;

describe("MultiCombobox filtering", () => {
  it("filters options based on input text", async () => {
    const user = userEvent.setup();

    render(
      <MultiCombobox options={options} displayValue={displayValue}>
        {(item) => <span>{item.name}</span>}
      </MultiCombobox>,
    );

    const input = screen.getByRole("combobox");

    await user.type(input, "app");

    expect(screen.getByText("Apple")).toBeDefined();
    expect(screen.queryByText("Banana")).toBeNull();
    expect(screen.queryByText("Lemon")).toBeNull();
  });
});
