import type { OrganizationsBrowserAction } from "@/api-client";

/**
 * Opens a provider browser action in the same tab. POST actions submit a
 * hidden form. GET actions assign the location.
 */
export function followBrowserAction(action: OrganizationsBrowserAction | undefined): boolean {
  if (!action?.url) return false;

  if (action.method?.toUpperCase() === "POST" && action.formFields) {
    const form = document.createElement("form");
    form.method = "POST";
    form.action = action.url;
    form.style.display = "none";
    Object.entries(action.formFields).forEach(([key, value]) => {
      const input = document.createElement("input");
      input.type = "hidden";
      input.name = key;
      input.value = String(value);
      form.appendChild(input);
    });
    document.body.appendChild(form);
    form.submit();
    return true;
  }

  window.location.assign(action.url);
  return true;
}
