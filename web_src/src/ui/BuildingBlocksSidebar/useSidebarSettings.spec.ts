import { renderHook, act } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { BUILDING_BLOCKS_SIDEBAR_SETTINGS_KEY, useSidebarSettings } from "./useSidebarSettings";

describe("useSidebarSettings", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("returns defaults when localStorage is empty", () => {
    const { result } = renderHook(() => useSidebarSettings());

    expect(result.current.showIntegrationSetupStatus).toBe(true);
    expect(result.current.showConnectedIntegrationsOnTop).toBe(false);
  });

  it("reads persisted values from localStorage on init", () => {
    window.localStorage.setItem(
      BUILDING_BLOCKS_SIDEBAR_SETTINGS_KEY,
      JSON.stringify({ showIntegrationSetupStatus: false, showConnectedIntegrationsOnTop: true }),
    );

    const { result } = renderHook(() => useSidebarSettings());

    expect(result.current.showIntegrationSetupStatus).toBe(false);
    expect(result.current.showConnectedIntegrationsOnTop).toBe(true);
  });

  it("writes to localStorage when a setting changes", () => {
    const { result } = renderHook(() => useSidebarSettings());

    act(() => {
      result.current.setShowConnectedIntegrationsOnTop(true);
    });

    const stored = JSON.parse(window.localStorage.getItem(BUILDING_BLOCKS_SIDEBAR_SETTINGS_KEY) || "{}");
    expect(stored.showConnectedIntegrationsOnTop).toBe(true);
    expect(stored.showIntegrationSetupStatus).toBe(true);
  });

  it("handles corrupted JSON gracefully", () => {
    window.localStorage.setItem(BUILDING_BLOCKS_SIDEBAR_SETTINGS_KEY, "not-json");

    const { result } = renderHook(() => useSidebarSettings());

    expect(result.current.showIntegrationSetupStatus).toBe(true);
    expect(result.current.showConnectedIntegrationsOnTop).toBe(false);
  });

  it("handles a stored array gracefully", () => {
    window.localStorage.setItem(BUILDING_BLOCKS_SIDEBAR_SETTINGS_KEY, "[1,2,3]");

    const { result } = renderHook(() => useSidebarSettings());

    expect(result.current.showIntegrationSetupStatus).toBe(true);
    expect(result.current.showConnectedIntegrationsOnTop).toBe(false);
  });

  it("fills missing fields with defaults", () => {
    window.localStorage.setItem(
      BUILDING_BLOCKS_SIDEBAR_SETTINGS_KEY,
      JSON.stringify({ showConnectedIntegrationsOnTop: true }),
    );

    const { result } = renderHook(() => useSidebarSettings());

    expect(result.current.showIntegrationSetupStatus).toBe(true);
    expect(result.current.showConnectedIntegrationsOnTop).toBe(true);
  });

  it("ignores non-boolean stored values and falls back to defaults", () => {
    window.localStorage.setItem(
      BUILDING_BLOCKS_SIDEBAR_SETTINGS_KEY,
      JSON.stringify({ showIntegrationSetupStatus: "yes", showConnectedIntegrationsOnTop: 1 }),
    );

    const { result } = renderHook(() => useSidebarSettings());

    expect(result.current.showIntegrationSetupStatus).toBe(true);
    expect(result.current.showConnectedIntegrationsOnTop).toBe(false);
  });
});
