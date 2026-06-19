import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  getStoredUtmAttribution,
  getUtmAttributionFromSearch,
  getUtmCookieDomain,
  getUtmEventProperties,
  initializeUtmAttribution,
} from "@/lib/utmAttribution";

const setOnce = vi.fn();
const posthog = {
  people: {
    set_once: setOnce,
  },
};

describe("utmAttribution", () => {
  beforeEach(() => {
    setOnce.mockClear();
    localStorage.clear();
    document.cookie = "superplane_initial_utm=; Max-Age=0; Path=/";
    window.history.replaceState({}, "", "/");
  });

  afterEach(() => {
    localStorage.clear();
    document.cookie = "superplane_initial_utm=; Max-Age=0; Path=/";
    window.history.replaceState({}, "", "/");
  });

  it("extracts supported UTM values from search params", () => {
    expect(
      getUtmAttributionFromSearch(
        "?utm_source=youtube&utm_medium=influencer&utm_campaign=erictech_beta&utm_content=video&ignored=value",
      ),
    ).toEqual({
      utm_source: "youtube",
      utm_medium: "influencer",
      utm_campaign: "erictech_beta",
      utm_content: "video",
    });
  });

  it("stores first-touch UTM values and sets PostHog person properties once", () => {
    window.history.replaceState(
      {},
      "",
      "/signup?utm_source=linkedin&utm_medium=influencer&utm_campaign=erictech_beta&utm_content=linkedin",
    );

    initializeUtmAttribution(posthog);

    expect(getStoredUtmAttribution()).toEqual({
      utm_source: "linkedin",
      utm_medium: "influencer",
      utm_campaign: "erictech_beta",
      utm_content: "linkedin",
    });
    expect(setOnce).toHaveBeenCalledWith({
      $initial_utm_source: "linkedin",
      $initial_utm_medium: "influencer",
      $initial_utm_campaign: "erictech_beta",
      $initial_utm_content: "linkedin",
    });
  });

  it("does not overwrite first-touch UTM values with later campaign params", () => {
    window.history.replaceState({}, "", "/?utm_source=youtube&utm_campaign=erictech_beta");
    initializeUtmAttribution(posthog);

    window.history.replaceState({}, "", "/?utm_source=linkedin&utm_campaign=later_campaign");
    initializeUtmAttribution(posthog);

    expect(getUtmEventProperties()).toEqual({
      utm_source: "youtube",
      utm_campaign: "erictech_beta",
    });
  });

  it("uses a shared parent-domain cookie on SuperPlane production domains", () => {
    expect(getUtmCookieDomain("superplane.com")).toBe(".superplane.com");
    expect(getUtmCookieDomain("app.superplane.com")).toBe(".superplane.com");
    expect(getUtmCookieDomain("localhost")).toBeUndefined();
  });
});
