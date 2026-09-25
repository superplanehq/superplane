import { describe, expect, it } from "bun:test";
import { mermaidInitializeConfig } from "./mermaidTheme";

describe("mermaidInitializeConfig", () => {
  it("keeps the current light palette and node fills", () => {
    const config = mermaidInitializeConfig("light");
    const vars = config.themeVariables ?? {};

    expect(config.theme).toBe("base");
    expect(vars.primaryColor).toBe("#ddd6fe");
    expect(vars.primaryTextColor).toBe("#4c1d95");
    expect(vars.primaryBorderColor).toBe("#7c3aed");
    expect(vars.secondaryColor).toBe("#67e8f9");
    expect(vars.secondaryTextColor).toBe("#164e63");
    expect(vars.secondaryBorderColor).toBe("#0891b2");
    expect(vars.tertiaryColor).toBe("#fcd34d");
    expect(vars.tertiaryTextColor).toBe("#78350f");
    expect(vars.tertiaryBorderColor).toBe("#d97706");
    expect(vars.lineColor).toBe("#64748b");
    expect(vars.textColor).toBe("#1e293b");
    expect(vars.nodeBorder).toBe("#7c3aed");
    expect(vars.nodeTextColor).toBe("#1e293b");
    expect(vars.clusterBkg).toBe("#f8fafc");
    expect(vars.clusterBorder).toBe("#cbd5e1");
    expect(vars.defaultLinkColor).toBe("#7c3aed");
    expect(vars.pie1).toBe("#8b5cf6");
    expect(vars.pieTitleTextColor).toBe("#1e293b");
    expect(vars.pieSectionTextColor).toBe("#0f172a");
    expect(vars.pieLegendTextColor).toBe("#334155");
  });

  it("uses light labels and lines on the dark card and keeps node fills", () => {
    const config = mermaidInitializeConfig("dark");
    const vars = config.themeVariables ?? {};

    expect(config.theme).toBe("base");
    expect(vars.primaryColor).toBe("#ddd6fe");
    expect(vars.primaryTextColor).toBe("#4c1d95");
    expect(vars.secondaryColor).toBe("#67e8f9");
    expect(vars.tertiaryColor).toBe("#fcd34d");
    expect(vars.nodeTextColor).toBe("#1e293b");
    expect(vars.pie1).toBe("#8b5cf6");
    expect(vars.pieSectionTextColor).toBe("#0f172a");

    expect(vars.textColor).toBe("#e2e8f0");
    expect(vars.lineColor).toBe("#cbd5e1");
    expect(vars.pieTitleTextColor).toBe("#e2e8f0");
    expect(vars.pieLegendTextColor).toBe("#e2e8f0");
    expect(vars.signalTextColor).toBe("#e2e8f0");
    expect(vars.labelTextColor).toBe("#e2e8f0");
    expect(vars.loopTextColor).toBe("#e2e8f0");
    expect(vars.sequenceNumberColor).toBe("#e2e8f0");
    expect(vars.actorLineColor).toBe("#cbd5e1");
    expect(vars.signalColor).toBe("#cbd5e1");
  });
});
