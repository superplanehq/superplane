import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";
import type { ConfigurationField } from "@/api-client";
import { SelectFieldRenderer } from "./SelectFieldRenderer";

describe("SelectFieldRenderer", () => {
  it("shows disabled options with their reason and prevents selection", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    const field: ConfigurationField = {
      name: "githubLocation",
      label: "Where is your GitHub?",
      type: "select",
      typeOptions: {
        select: {
          options: [
            { label: "GitHub.com", value: "github.com" },
            {
              label: "GitHub Enterprise Cloud",
              value: "enterprise-cloud",
              disabled: true,
              description: "This option needs a SuperPlane Enterprise license.",
            },
          ],
        },
      },
    };

    render(<SelectFieldRenderer field={field} value="github.com" onChange={onChange} />);
    await user.click(screen.getByRole("combobox"));
    const enterprise = screen.getByRole("option", { name: /GitHub Enterprise Cloud/ });
    expect(enterprise.getAttribute("aria-disabled")).toBe("true");
    expect(screen.getByText("This option needs a SuperPlane Enterprise license.")).toBeVisible();
    await user.click(enterprise);
    expect(onChange).not.toHaveBeenCalled();
    await user.keyboard("{ArrowDown}{Enter}");
    expect(onChange).not.toHaveBeenCalledWith("enterprise-cloud");
  });
});
