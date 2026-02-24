import type { SetupActionRedirect } from "@/api-client";

export const executeSetupRedirectAction = (redirect?: SetupActionRedirect | null) => {
  const url = redirect?.url;
  if (!url) return;

  if (redirect.method?.toUpperCase() === "POST" && redirect.formFields) {
    const form = document.createElement("form");
    form.method = "POST";
    form.action = url;
    form.target = "_blank";
    form.style.display = "none";

    Object.entries(redirect.formFields).forEach(([key, value]) => {
      const input = document.createElement("input");
      input.type = "hidden";
      input.name = key;
      input.value = String(value);
      form.appendChild(input);
    });

    document.body.appendChild(form);
    form.submit();
    document.body.removeChild(form);
    return;
  }

  window.open(url, "_blank");
};
