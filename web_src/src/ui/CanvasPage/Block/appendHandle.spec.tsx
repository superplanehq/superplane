import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AppendHandlePreview } from "./appendHandle";

describe("AppendHandlePreview", () => {
  it("matches factory node dimensions for vertical canvases", () => {
    const { container } = render(<AppendHandlePreview orientation="vertical" style={{}} />);

    const previewCard = container.querySelector<HTMLElement>(".sp-append-source-preview > div:last-child");

    expect(previewCard?.style.width).toBe("280px");
    expect(previewCard?.style.height).toBe("104px");
    expect(previewCard?.style.borderRadius).toBe("16px");
  });

  it("keeps legacy dimensions for horizontal canvases", () => {
    const { container } = render(<AppendHandlePreview style={{}} />);

    const previewCard = container.querySelector<HTMLElement>(".sp-append-source-preview > div:last-child");

    expect(previewCard?.style.width).toBe("23rem");
    expect(previewCard?.style.height).toBe("96px");
    expect(previewCard?.style.borderRadius).toBe("8px");
  });
});
