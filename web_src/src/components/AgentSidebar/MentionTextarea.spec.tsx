import { createRef } from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MentionTextarea } from "./MentionTextarea";
import type { InsertedMention } from "./useMentions";

function renderTextarea(value = "") {
  const textareaRef = createRef<HTMLTextAreaElement>();
  const backdropRef = createRef<HTMLDivElement>();
  render(
    <MentionTextarea
      value={value}
      mentions={[]}
      setValue={vi.fn()}
      setCursorPos={vi.fn()}
      onKeyDown={vi.fn()}
      textareaRef={textareaRef}
      backdropRef={backdropRef}
    />,
  );

  return {
    textarea: screen.getByTestId("agent-input") as HTMLTextAreaElement,
    backdrop: backdropRef.current as HTMLDivElement,
  };
}

function setReadonlyNumber(element: HTMLElement, property: "clientHeight" | "scrollHeight", value: number) {
  Object.defineProperty(element, property, { configurable: true, value });
}

describe("MentionTextarea", () => {
  it("grows to fit multiline content before internal scrolling", () => {
    const { textarea } = renderTextarea();
    setReadonlyNumber(textarea, "scrollHeight", 72);

    fireEvent.change(textarea, { target: { value: "@New Component\nasdasdasdas" } });

    expect(textarea.style.height).toBe("72px");
    expect(textarea.style.overflowY).toBe("hidden");
  });

  it("caps height and syncs internal scroll with the backdrop", () => {
    const mention: InsertedMention = {
      id: "node-1",
      type: "node",
      label: "New Component",
      displayText: "@New Component",
      startIndex: 0,
    };
    const textareaRef = createRef<HTMLTextAreaElement>();
    const backdropRef = createRef<HTMLDivElement>();
    render(
      <MentionTextarea
        value="@New Component\none\ntwo\nthree\nfour\nfive\nsix"
        mentions={[mention]}
        setValue={vi.fn()}
        setCursorPos={vi.fn()}
        onKeyDown={vi.fn()}
        textareaRef={textareaRef}
        backdropRef={backdropRef}
      />,
    );

    const textarea = screen.getByTestId("agent-input") as HTMLTextAreaElement;
    const backdrop = backdropRef.current as HTMLDivElement;
    setReadonlyNumber(textarea, "scrollHeight", 240);

    fireEvent.change(textarea, { target: { value: `${textarea.value}\nseven` } });
    textarea.scrollTop = 56;
    fireEvent.scroll(textarea);

    expect(textarea.style.height).toBe("144px");
    expect(textarea.style.overflowY).toBe("auto");
    expect(backdrop.scrollTop).toBe(56);
  });
});
