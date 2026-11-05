import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, it, vi } from "bun:test";
import { Button } from "@/components/ui/button";
import field from "@/assets/superplane-auth-field.svg";
import { LoginLayout } from "./LoginLayout";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  delete window.THREE;
  delete window.SuperplaneArt;
});

it("keeps static artwork and an interactive login control when eligible animation mounting fails", async () => {
  const media = window.matchMedia("(min-width: 851px) and (prefers-reduced-motion: no-preference)");
  vi.spyOn(window, "matchMedia").mockReturnValue({
    ...media,
    matches: true,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  });
  const mount = vi.fn((target: HTMLElement) => {
    target.append(document.createElement("canvas"));
    throw new Error("renderer initialization failed");
  });
  window.THREE = {};
  window.SuperplaneArt = { mount };
  const submit = vi.fn();
  const view = render(
    <LoginLayout heading="Log in" subheading="Welcome back">
      <Button onClick={submit}>Log in</Button>
    </LoginLayout>,
  );

  await waitFor(() => expect(mount).toHaveBeenCalledTimes(1));
  expect(screen.getByTestId("first-run-art-canvas").childNodes).toHaveLength(0);
  const fallback = Array.from(view.container.querySelectorAll("img")).find(
    (image) => image.getAttribute("src") === field,
  );
  expect(fallback).toBeVisible();
  const control = screen.getByRole("button", { name: "Log in" });
  expect(control).toBeEnabled();
  fireEvent.click(control);
  expect(submit).toHaveBeenCalledTimes(1);
});
