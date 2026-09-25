import { beforeEach, afterEach, describe, expect, it } from "bun:test";
import { initGoogleTagManager, trackGoogleSignup, trackGooglePurchase, confirmedPurchase } from "./googleTagManager";

const invoice = {
  id: "order-1",
  checkoutId: "checkout-1",
  status: "paid",
  currency: "usd",
  netAmountCents: "9000",
  taxAmountCents: "1000",
  productName: "Business",
};

describe("Google Tag Manager", () => {
  afterEach(() => {
    Reflect.deleteProperty(window.location, "hostname");
  });
  beforeEach(() => {
    Object.defineProperty(window.location, "hostname", { configurable: true, value: "app.superplane.com" });
    delete window.dataLayer;
    document.getElementById("superplane-gtm")?.remove();
    localStorage.clear();
  });

  it("preserves queued data and initializes the shared container once", () => {
    window.dataLayer = [{ event: "queued" }];
    initGoogleTagManager();
    initGoogleTagManager();
    expect(window.dataLayer.map((event) => event.event)).toEqual(["queued", "gtm.js"]);
    expect(document.querySelectorAll("#superplane-gtm")).toHaveLength(1);
    expect((document.getElementById("superplane-gtm") as HTMLScriptElement).src).toBe(
      "https://www.googletagmanager.com/gtm.js?id=GTM-TKMMDB5T",
    );
  });

  it.each(["localhost", "preview.superplane.com", "self-hosted.example", "app.superplane.com.example.com"])(
    "does not load or track on %s",
    (hostname) => {
      Object.defineProperty(window.location, "hostname", { configurable: true, value: hostname });
      initGoogleTagManager();
      trackGoogleSignup(`nonproduction-${hostname}`);
      trackGooglePurchase("checkout-1", [invoice]);
      expect(window.dataLayer).toBeUndefined();
      expect(document.getElementById("superplane-gtm")).toBeNull();
    },
  );

  it("deduplicates signup without sending account details", () => {
    trackGoogleSignup("signup-1");
    trackGoogleSignup("signup-1");
    expect(window.dataLayer).toEqual([{ event: "sign_up" }]);
  });

  it("honors persisted signup deduplication", () => {
    localStorage.setItem("superplane:gtm:sign_up:previous-account", "1");
    trackGoogleSignup("previous-account");
    expect(window.dataLayer).toBeUndefined();
  });

  it("emits a paid checkout with its transaction ID, currency, net value, and tax once", () => {
    expect(trackGooglePurchase("checkout-1", [invoice])).toBe(true);
    trackGooglePurchase("checkout-1", [invoice]);
    expect(window.dataLayer).toEqual([
      { ecommerce: null },
      {
        event: "purchase",
        ecommerce: {
          transaction_id: "order-1",
          currency: "USD",
          value: 90,
          tax: 10,
          items: [{ item_name: "Business", price: 90, quantity: 1 }],
        },
      },
    ]);
  });

  it("rejects missing, unrelated, unpaid, and malformed purchases", () => {
    expect(confirmedPurchase("", [invoice])).toBeUndefined();
    expect(confirmedPurchase("other-checkout", [invoice])).toBeUndefined();
    for (const status of ["pending", "refunded", "void"]) {
      expect(confirmedPurchase("checkout-1", [{ ...invoice, status }])).toBeUndefined();
    }
    for (const netAmountCents of [undefined, "NaN", "-1"]) {
      expect(confirmedPurchase("checkout-1", [{ ...invoice, netAmountCents }])).toBeUndefined();
    }
    expect(confirmedPurchase("checkout-1", [{ ...invoice, currency: "" }])).toBeUndefined();
    expect(trackGooglePurchase("checkout-1", [])).toBe(false);
    expect(window.dataLayer).toBeUndefined();
  });
});
