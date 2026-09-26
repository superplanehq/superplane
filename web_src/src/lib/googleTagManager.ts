const productionHostname = "app.superplane.com";
const containerID = "GTM-TKMMDB5T";
const emittedEvents = new Set<string>();

declare global {
  interface Window {
    dataLayer?: Record<string, unknown>[];
  }
}

export function isGoogleTagManagerEnabled() {
  return window.location.hostname === productionHostname;
}

export function initGoogleTagManager() {
  if (!isGoogleTagManagerEnabled() || document.getElementById("superplane-gtm")) return;
  window.dataLayer = window.dataLayer || [];
  window.dataLayer.push({ "gtm.start": Date.now(), event: "gtm.js" });
  const script = document.createElement("script");
  script.id = "superplane-gtm";
  script.async = true;
  script.src = `https://www.googletagmanager.com/gtm.js?id=${containerID}`;
  document.head.appendChild(script);
}

function pushOnce(key: string, event: Record<string, unknown>) {
  if (!isGoogleTagManagerEnabled() || emittedEvents.has(key)) return;
  const storageKey = `superplane:gtm:${key}`;
  try {
    if (localStorage.getItem(storageKey)) return;
  } catch {
    // Tracking must not interrupt authentication or billing when storage is unavailable.
  }
  window.dataLayer = window.dataLayer || [];
  if (event.event === "purchase") window.dataLayer.push({ ecommerce: null });
  window.dataLayer.push(event);
  emittedEvents.add(key);
  try {
    localStorage.setItem(storageKey, "1");
  } catch {
    // The in-memory set still prevents duplicate events in this page session.
  }
}

export function trackGoogleSignup(accountID: string) {
  if (accountID) pushOnce(`sign_up:${accountID}`, { event: "sign_up" });
}

export type PurchaseInvoice = {
  id?: string;
  checkoutId?: string;
  status?: string;
  currency?: string;
  netAmountCents?: string;
  taxAmountCents?: string;
  productName?: string;
};

export function confirmedPurchase(checkoutID: string, invoices: PurchaseInvoice[]) {
  if (!checkoutID) return undefined;
  return invoices.find((invoice) => {
    const value = Number(invoice.netAmountCents);
    const tax = Number(invoice.taxAmountCents);
    return (
      invoice.checkoutId === checkoutID &&
      invoice.status === "paid" &&
      Boolean(invoice.id) &&
      /^[a-z]{3}$/i.test(invoice.currency ?? "") &&
      invoice.netAmountCents !== undefined &&
      invoice.taxAmountCents !== undefined &&
      Number.isFinite(value) &&
      value >= 0 &&
      Number.isFinite(tax) &&
      tax >= 0
    );
  });
}

export function trackGooglePurchase(checkoutID: string, invoices: PurchaseInvoice[]) {
  const invoice = confirmedPurchase(checkoutID, invoices);
  if (!invoice) return false;
  pushOnce(`purchase:${invoice.id}`, {
    event: "purchase",
    ecommerce: {
      transaction_id: invoice.id,
      currency: invoice.currency!.toUpperCase(),
      value: Number(invoice.netAmountCents) / 100,
      tax: Number(invoice.taxAmountCents) / 100,
      items: [
        { item_name: invoice.productName || "SuperPlane", price: Number(invoice.netAmountCents) / 100, quantity: 1 },
      ],
    },
  });
  return true;
}
