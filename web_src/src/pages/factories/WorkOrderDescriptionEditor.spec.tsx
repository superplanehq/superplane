import { fireEvent, render, screen } from "@testing-library/react";
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

  it("keeps the format toolbar visible when the heading menu opens", async () => {
    const user = userEvent.setup();

    render(<WorkOrderDescriptionEditor value="Hello world" maxLength={5000} disabled={false} onChange={vi.fn()} />);

    const input = await screen.findByTestId("work-order-description-input");
    await user.click(input);
    await user.keyboard("{Control>}a{/Control}");
    await user.click(await screen.findByRole("button", { name: "Heading" }));

    expect(screen.getByTestId("work-order-description-toolbar")).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "Heading 1" })).toBeInTheDocument();
  });

  it("keeps existing description text when the length limit is reached", async () => {
    const user = userEvent.setup();
    const initial = "Hello world";
    const onChange = vi.fn();

    render(
      <WorkOrderDescriptionEditor value={initial} maxLength={initial.length} disabled={false} onChange={onChange} />,
    );

    const input = await screen.findByTestId("work-order-description-input");
    await user.click(input);
    await user.keyboard(" extra");

    expect(input.textContent).toContain(initial);
    expect(input.textContent).not.toContain("extra");
    expect(onChange).not.toHaveBeenCalled();
  });

  it("keeps the first characters of a long pasted description", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    const maxLength = 20;

    render(<WorkOrderDescriptionEditor value="" maxLength={maxLength} disabled={false} onChange={onChange} />);

    const input = await screen.findByTestId("work-order-description-input");
    await user.click(input);
    await user.paste("a".repeat(40));

    expect(input.textContent).toMatch(/^a+$/);
    expect(input.textContent?.length).toBe(maxLength);
    expect(onChange).toHaveBeenCalled();
    expect(onChange.mock.calls.at(-1)?.[0]).toHaveLength(maxLength);
  });

  it("does not drop description text when formatting at the length limit", async () => {
    const user = userEvent.setup();
    const initial = "Hello world";
    const onChange = vi.fn();

    render(
      <WorkOrderDescriptionEditor value={initial} maxLength={initial.length} disabled={false} onChange={onChange} />,
    );

    const input = await screen.findByTestId("work-order-description-input");
    await user.click(input);
    await user.keyboard("{Control>}a{/Control}");
    await user.click(await screen.findByRole("button", { name: "Bold" }));

    expect(input.textContent).toContain(initial);
    expect(onChange).not.toHaveBeenCalled();
  });

  it("does not drop description text after a no-op paste at the length limit", async () => {
    const user = userEvent.setup();
    const initial = "Hello world";
    const onChange = vi.fn();

    render(
      <WorkOrderDescriptionEditor value={initial} maxLength={initial.length} disabled={false} onChange={onChange} />,
    );

    const input = await screen.findByTestId("work-order-description-input");
    await user.click(input);
    fireEvent.paste(input, {
      clipboardData: { getData: () => "" },
    });
    await user.keyboard("{Control>}a{/Control}");
    await user.click(await screen.findByRole("button", { name: "Bold" }));

    expect(input.textContent).toContain(initial);
    expect(onChange).not.toHaveBeenCalled();
  });

  it("underlines selected text and stores it as HTML", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();

    render(<WorkOrderDescriptionEditor value="Hello world" maxLength={5000} disabled={false} onChange={onChange} />);

    const input = await screen.findByTestId("work-order-description-input");
    await user.click(input);
    await user.keyboard("{Control>}a{/Control}");
    await user.click(await screen.findByRole("button", { name: "Underline" }));

    expect(input.querySelector("u")).toHaveTextContent("Hello world");
    expect(onChange.mock.calls.at(-1)?.[0]).toContain("<u>Hello world</u>");
  });

  it("renders stored underline HTML when the editor opens", async () => {
    render(
      <WorkOrderDescriptionEditor value="Hello <u>world</u>." maxLength={5000} disabled={false} onChange={vi.fn()} />,
    );

    const input = await screen.findByTestId("work-order-description-input");
    expect(input.querySelector("u")).toHaveTextContent("world");
  });

  it("does not treat increment operators as underline when the editor opens", async () => {
    render(<WorkOrderDescriptionEditor value="i++ then j++" maxLength={5000} disabled={false} onChange={vi.fn()} />);

    const input = await screen.findByTestId("work-order-description-input");
    expect(input.querySelector("u")).toBeNull();
    expect(input).toHaveTextContent("i++ then j++");
  });

  it("applies bold from the format toolbar with the keyboard", async () => {
    const user = userEvent.setup();

    render(<WorkOrderDescriptionEditor value="Hello world" maxLength={5000} disabled={false} onChange={vi.fn()} />);

    const input = await screen.findByTestId("work-order-description-input");
    await user.click(input);
    await user.keyboard("{Control>}a{/Control}");
    const bold = await screen.findByRole("button", { name: "Bold" });
    bold.focus();
    await user.keyboard("{Enter}");

    expect(input.querySelector("strong")).toHaveTextContent("Hello world");
  });
});
