import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "bun:test";

import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from "@/ui/dropdownMenu";

import { DropdownMenuValueSub } from "./DropdownMenuValueSub";

beforeAll(() => {
  Element.prototype.hasPointerCapture ??= () => false;
  Element.prototype.setPointerCapture ??= () => {};
  Element.prototype.releasePointerCapture ??= () => {};
  Element.prototype.scrollIntoView ??= () => {};
});

describe("DropdownMenuValueSub", () => {
  it("caps the fly-out height so a long option list can scroll", async () => {
    const user = userEvent.setup();
    const onValueChange = vi.fn();
    const options = Array.from({ length: 24 }, (_, index) => ({
      value: `model-${index}`,
      label: `model-${index}`,
    }));

    render(
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button type="button">Open</Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuValueSub
            label="Model"
            testId="model-flyout"
            value="model-0"
            options={options}
            onValueChange={onValueChange}
          />
        </DropdownMenuContent>
      </DropdownMenu>,
    );

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.hover(screen.getByTestId("model-flyout"));
    const option = await screen.findByRole("menuitem", { name: "model-23" });
    expect(option.parentElement).toHaveClass("overflow-y-auto");
    expect(option.parentElement?.className).toContain("max-h-[var(--radix-dropdown-menu-content-available-height)]");
    fireEvent.click(option);
    expect(onValueChange).toHaveBeenCalledWith("model-23");
  });
});
