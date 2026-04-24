import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ConfigurationField } from "@/api-client";
import { MultiSelectFieldRenderer } from "./MultiSelectFieldRenderer";

function createField(useCheckboxes: boolean): ConfigurationField {
  return {
    name: "deliveryChannels",
    label: "Delivery channels",
    type: "multi-select",
    typeOptions: {
      multiSelect: {
        useCheckboxes,
        options: [
          {
            label: "Email",
            value: "email",
            description: "Send notifications by email.",
          },
          {
            label: "Slack",
            value: "slack",
            description: "Post notifications to a Slack channel.",
          },
        ],
      },
    },
  };
}

describe("MultiSelectFieldRenderer", () => {
  it("renders checkbox options with descriptions when useCheckboxes is enabled", () => {
    render(<MultiSelectFieldRenderer field={createField(true)} value={["email"]} onChange={vi.fn()} />);

    expect(screen.getByRole("checkbox", { name: /Email/ })).toHaveAttribute("aria-checked", "true");
    expect(screen.getByRole("checkbox", { name: /Slack/ })).toHaveAttribute("aria-checked", "false");
    expect(screen.getByText("Send notifications by email.")).toBeInTheDocument();
    expect(screen.getByText("Post notifications to a Slack channel.")).toBeInTheDocument();
  });

  it("adds values when a checkbox is checked", async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();

    render(<MultiSelectFieldRenderer field={createField(true)} value={[]} onChange={handleChange} />);

    await user.click(screen.getByRole("checkbox", { name: /Email/ }));

    expect(handleChange).toHaveBeenCalledWith(["email"]);
  });

  it("sends undefined when the last selected value is unchecked", async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();

    render(<MultiSelectFieldRenderer field={createField(true)} value={["email"]} onChange={handleChange} />);

    await user.click(screen.getByRole("checkbox", { name: /Email/ }));

    expect(handleChange).toHaveBeenCalledWith(undefined);
  });
});
