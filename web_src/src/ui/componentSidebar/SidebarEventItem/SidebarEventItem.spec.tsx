import type { CSSProperties } from "react";
import React from "react";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { darkJsonViewStyle, lightJsonViewStyle } from "@/lib/jsonViewTheme";
import type { SidebarEvent } from "../types";
import { SidebarEventItem } from "./SidebarEventItem";

const { jsonViewStyles, themeState } = vi.hoisted(() => ({
  jsonViewStyles: [] as CSSProperties[],
  themeState: { resolvedTheme: "light" as "light" | "dark" },
}));

vi.mock("@uiw/react-json-view", () => ({
  default: ({ style }: { style?: CSSProperties }) => {
    jsonViewStyles.push(style ?? {});
    return <div data-testid="json-view" />;
  },
}));

vi.mock("@/contexts/useTheme", () => ({
  useTheme: () => ({
    preference: themeState.resolvedTheme,
    resolvedTheme: themeState.resolvedTheme,
    setPreference: () => undefined,
  }),
}));

vi.mock("@tippyjs/react/headless", () => ({
  default: ({ children }: { children: React.ReactNode }) => children,
}));

const event: SidebarEvent = {
  id: "event-1",
  title: "HTTP Request",
  subtitle: "GET",
  state: "success",
};

const payload = { status: 200, body: { ok: true } };

function renderPayloadItem() {
  return render(<SidebarEventItem event={event} index={0} isOpen onToggleOpen={vi.fn()} tabData={{ payload }} />);
}

describe("SidebarEventItem payload JSON theme", () => {
  beforeEach(() => {
    jsonViewStyles.length = 0;
    themeState.resolvedTheme = "light";
  });

  it("applies the dark jsonViewTheme to the payload viewer", () => {
    themeState.resolvedTheme = "dark";
    renderPayloadItem();

    expect(screen.getByTestId("json-view")).toBeInTheDocument();
    expect(jsonViewStyles[0]).toMatchObject({
      backgroundColor: darkJsonViewStyle.backgroundColor,
      color: darkJsonViewStyle.color,
    });
  });

  it("applies the light jsonViewTheme to the payload viewer", () => {
    renderPayloadItem();

    expect(jsonViewStyles[0]).toMatchObject({
      backgroundColor: lightJsonViewStyle.backgroundColor,
      color: lightJsonViewStyle.color,
    });
  });
});
