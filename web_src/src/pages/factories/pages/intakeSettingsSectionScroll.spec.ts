import { describe, expect, it } from "bun:test";

import {
  intakeSettingsActiveSectionId,
  intakeSettingsClickScrollTop,
  type IntakeSettingsScrollMetrics,
  type IntakeSettingsSectionOffset,
} from "./intakeSettingsSectionScroll";

const connection = { id: "connection", top: 24 };
const triggers = { id: "triggers", top: 80 };
const labels = { id: "labels", top: 400 };
const factory = { id: "factory", top: 700 };
const danger = { id: "danger", top: 1300 };

const jiraSections: IntakeSettingsSectionOffset[] = [connection, triggers, labels, factory, danger];

function overflowingPane(scrollTop: number): IntakeSettingsScrollMetrics {
  return { scrollTop, clientHeight: 400, scrollHeight: 1400, paddingTop: 24 };
}

describe("intakeSettingsActiveSectionId", () => {
  it("returns null when there are no sections", () => {
    expect(intakeSettingsActiveSectionId(overflowingPane(0), [])).toBeNull();
  });

  it("returns null when content does not overflow so the caller keeps the current highlight", () => {
    expect(
      intakeSettingsActiveSectionId(
        { scrollTop: 0, clientHeight: 800, scrollHeight: 400, paddingTop: 24 },
        jiraSections,
      ),
    ).toBeNull();
  });

  it("selects the first section at scroll 0 even when a later section sits in the old observer band", () => {
    expect(intakeSettingsActiveSectionId(overflowingPane(0), jiraSections)).toBe("connection");
  });

  it("selects the section whose heading sits on the spy line", () => {
    expect(intakeSettingsActiveSectionId(overflowingPane(labels.top - 24), jiraSections)).toBe("labels");
  });

  it("selects the last section near the bottom", () => {
    expect(intakeSettingsActiveSectionId(overflowingPane(999), jiraSections)).toBe("danger");
  });
});

describe("intakeSettingsClickScrollTop", () => {
  it("scrolls the first section to the start of the pane", () => {
    expect(intakeSettingsClickScrollTop({ top: 24, isFirst: true }, 24)).toBe(0);
  });

  it("aligns later sections with the padded heading position", () => {
    expect(intakeSettingsClickScrollTop({ top: 400, isFirst: false }, 24)).toBe(376);
  });
});
