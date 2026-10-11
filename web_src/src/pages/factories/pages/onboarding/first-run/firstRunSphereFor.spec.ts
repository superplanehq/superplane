import { describe, expect, it } from "bun:test";

import { analysisSphereFor, repositorySphereFor, sphereFor } from "./firstRunSphereFor";

describe("sphereFor", () => {
  it("maps each onboarding step onto the preview globe", () => {
    expect(sphereFor("welcome", null).art).toMatchObject({
      mode: "school",
      background: "#87ae9d",
      arrowColor: "#eeede9",
      pill: { label: "Discover", value: "Awaiting code", light: false },
    });
    expect(sphereFor("welcome", null).art?.count).toBeUndefined();

    expect(sphereFor("host", null).art).toMatchObject({
      mode: "globe",
      background: "#b7b174",
      arrowColor: "#eeede9",
      count: 100,
    });
    expect(sphereFor("connect", null).art).toEqual(sphereFor("host", null).art);

    expect(sphereFor("choose", null).art).toMatchObject({
      mode: "globe",
      background: "#b09532",
      arrowColor: "#eeede9",
      count: 400,
    });

    expect(repositorySphereFor("acme").art).toMatchObject({
      mode: "globe",
      background: "#c5ebc3",
      arrowColor: "#11110e",
      count: 700,
      pill: { value: "acme", light: true },
    });
    expect(sphereFor("tickets", "acme/api", "acme").art).toEqual(repositorySphereFor("acme/api").art);
    expect(sphereFor("agent", null, "acme").art).toEqual(repositorySphereFor("acme").art);

    expect(analysisSphereFor("acme/api", 0).art).toMatchObject({
      mode: "globe",
      background: "#EF8D0B",
      arrowColor: "#11110e",
      count: 1000,
      badges: { discover: "acme/api", verify: "Review-ready PR" },
    });
    expect(analysisSphereFor("acme/api", 0).art?.pill).toBeUndefined();
  });
});
