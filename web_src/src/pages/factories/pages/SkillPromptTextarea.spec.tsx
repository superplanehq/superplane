import { describe, expect, it } from "bun:test";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { useState } from "react";
import { INLINE_SKILL } from "../__fixtures__/agentResourceFixtures";
import { SkillPromptTextarea } from "./SkillPromptTextarea";

function Editor({ initial = "" }: { initial?: string }) {
  const [value, setValue] = useState(initial);
  return (
    <MemoryRouter>
      <SkillPromptTextarea
        value={value}
        onChange={setValue}
        skills={[INLINE_SKILL]}
        label="Prompt"
        testId="prompt"
        skillHref={(skill) => `/skills/${skill.id}`}
      />
    </MemoryRouter>
  );
}

describe("skill prompt input", () => {
  it("filters commands, inserts a skill with the keyboard, and keeps plain text around the token", async () => {
    const user = userEvent.setup();
    render(<Editor initial="Use  to review." />);
    const field = screen.getByRole("textbox", { name: "Prompt" });
    await user.click(field);
    fireEvent.change(field, { target: { value: "Use /rev to review.", selectionStart: 8 } });
    expect(screen.getByRole("option", { name: /\/review-copy/ })).toBeInTheDocument();
    await user.keyboard("{Enter}");
    await waitFor(() => expect(field).toHaveValue("Use /review-copy to review."));
    expect(document.querySelector('[data-skill-token="review-copy"]')).toHaveTextContent("/review-copy");
  });

  it("allows mouse selection without losing the cursor or surrounding prompt text", async () => {
    const user = userEvent.setup();
    render(<Editor />);
    await user.type(screen.getByRole("textbox"), "Use /");
    await user.click(screen.getByRole("option", { name: /\/review-copy/ }));
    expect(screen.getByRole("textbox")).toHaveValue("Use /review-copy ");
    await user.keyboard("next");
    expect(screen.getByRole("textbox")).toHaveValue("Use /review-copy next");
  });

  it("does not treat URLs or paths as skill commands and leaves unknown names unstyled", () => {
    render(<Editor initial="https://example.com/review-copy /review-copy/path /unknown (/review-copy)." />);
    const field = screen.getByRole("textbox");
    fireEvent.focus(field);
    fireEvent.select(field, { target: { selectionStart: 31 } });
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    expect(document.querySelectorAll("[data-skill-token]")).toHaveLength(1);
    expect(field).toHaveValue("https://example.com/review-copy /review-copy/path /unknown (/review-copy).");
  });

  it("dismisses suggestions with Escape without preventing normal prompt editing", async () => {
    const user = userEvent.setup();
    render(<Editor />);
    await user.type(screen.getByRole("textbox"), "/rev");
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    await user.keyboard("{Enter}text");
    expect(screen.getByRole("textbox")).toHaveValue("/rev\ntext");
  });
  it("keeps skill tokens identifiable and returns editing to the prompt when clicked", async () => {
    const user = userEvent.setup();
    render(<Editor initial="Use /review-copy" />);
    const token = document.querySelector<HTMLElement>('[data-skill-token="review-copy"]')!;
    expect(token).toHaveAttribute("data-skill-id", INLINE_SKILL.id);
    expect(token).toHaveAttribute("tabindex", "-1");
    await user.click(token);
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    await waitFor(() => expect((screen.getByRole("textbox") as HTMLTextAreaElement).selectionStart).toBe(17));
    await user.keyboard("next");
    expect(screen.getByRole("textbox")).toHaveValue("Use /review-copy next");
  });
  it("shows a saved skill token's title, description and editor link on hover", async () => {
    const user = userEvent.setup();
    render(<Editor initial="Use /review-copy." />);
    await user.hover(document.querySelector<HTMLElement>('[data-skill-token="review-copy"]')!);
    expect(await screen.findByText("Review copy")).toBeInTheDocument();
    expect(screen.getByText("Review UI copy.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open skill" })).toHaveAttribute("href", `/skills/${INLINE_SKILL.id}`);
    expect(screen.getByRole("link", { name: "Open skill" })).toHaveAttribute("target", "_blank");
  });
  it("places the caret beyond an existing separator without changing the prompt", async () => {
    const user = userEvent.setup();
    render(<Editor initial="Use /review-copy next." />);
    await user.click(document.querySelector<HTMLElement>('[data-skill-token="review-copy"]')!);
    const field = screen.getByRole("textbox") as HTMLTextAreaElement;
    await waitFor(() => expect(field.selectionStart).toBe(17));
    expect(field).toHaveValue("Use /review-copy next.");
    await user.keyboard("the ");
    expect(field).toHaveValue("Use /review-copy the next.");
  });
});
