import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { WorkOrderDescriptionEditor } from "./WorkOrderDescriptionEditor";

const PASTED_MARKDOWN = `## Papercuts
These are small improvements or issues that improve quality of life.

- I don’t see anywhere which models are used for what. I don’t know how to find this info. 
- The branch artifact should be a link I can click on, same as PR.`;

const emptyRect = {
  x: 0,
  y: 0,
  width: 0,
  height: 0,
  top: 0,
  right: 0,
  bottom: 0,
  left: 0,
  toJSON() {
    return this;
  },
};
const emptyRects = {
  item: () => null,
  length: 0,
  [Symbol.iterator]: function* () {},
};

beforeAll(() => {
  document.elementFromPoint = () => null;
  stubClientRects(Range.prototype);
  stubClientRects(Element.prototype);
  stubClientRects(Text.prototype);
});

function stubClientRects(target: object) {
  Object.defineProperty(target, "getBoundingClientRect", {
    configurable: true,
    value: () => emptyRect,
  });
  Object.defineProperty(target, "getClientRects", {
    configurable: true,
    value: () => emptyRects,
  });
}

describe("WorkOrderDescriptionEditor", () => {
  it("renders pasted markdown as a heading, paragraph, and list", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();

    render(
      <WorkOrderDescriptionEditor
        value=""
        maxLength={5000}
        disabled={false}
        onChange={onChange}
        onFocus={vi.fn()}
        onBlur={vi.fn()}
      />,
    );

    const input = await screen.findByTestId("work-order-description-input");
    await user.click(input);
    await user.paste(PASTED_MARKDOWN);

    expect(screen.getByRole("heading", { level: 2, name: "Papercuts" })).toBeInTheDocument();
    expect(screen.queryByText("## Papercuts")).not.toBeInTheDocument();
    expect(
      screen.getByText("These are small improvements or issues that improve quality of life."),
    ).toBeInTheDocument();
    expect(screen.getAllByRole("listitem")).toHaveLength(2);
    expect(onChange).toHaveBeenCalledWith(expect.stringContaining("## Papercuts"));
  });

  it("turns a typed heading shortcut into a heading", async () => {
    const user = userEvent.setup();

    render(
      <WorkOrderDescriptionEditor
        value=""
        maxLength={5000}
        disabled={false}
        onChange={vi.fn()}
        onFocus={vi.fn()}
        onBlur={vi.fn()}
      />,
    );

    const input = await screen.findByTestId("work-order-description-input");
    await user.click(input);
    await user.keyboard("## Hello");

    expect(screen.getByRole("heading", { level: 2, name: "Hello" })).toBeInTheDocument();
    expect(screen.queryByText("## Hello")).not.toBeInTheDocument();
  });

  it("hides the format menu until text is selected", async () => {
    const user = userEvent.setup();

    render(<WorkOrderDescriptionEditor value="Hello world" maxLength={5000} disabled={false} onChange={vi.fn()} />);

    const input = await screen.findByTestId("work-order-description-input");
    await user.click(input);

    expect(screen.queryByTestId("work-order-description-toolbar")).not.toBeInTheDocument();

    await user.keyboard("{Control>}a{/Control}");

    expect(await screen.findByTestId("work-order-description-toolbar")).toBeInTheDocument();
  });

  it("shows heading, code block, quote, link, underline, and strikethrough controls", async () => {
    const user = userEvent.setup();

    render(<WorkOrderDescriptionEditor value="Hello world" maxLength={5000} disabled={false} onChange={vi.fn()} />);

    const input = await screen.findByTestId("work-order-description-input");
    await user.click(input);
    await user.keyboard("{Control>}a{/Control}");

    expect(await screen.findByTestId("work-order-description-toolbar")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Heading" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Code block" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Quote" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Link" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Underline" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Strikethrough" })).toBeInTheDocument();
  });

  it("turns selected text into a heading from the heading menu", async () => {
    const user = userEvent.setup();

    render(<WorkOrderDescriptionEditor value="Hello world" maxLength={5000} disabled={false} onChange={vi.fn()} />);

    const input = await screen.findByTestId("work-order-description-input");
    await user.click(input);
    await user.keyboard("{Control>}a{/Control}");
    await user.click(await screen.findByRole("button", { name: "Heading" }));
    await user.click(screen.getByRole("menuitem", { name: "Heading 1" }));

    expect(screen.getByRole("heading", { level: 1, name: "Hello world" })).toBeInTheDocument();
  });
});
