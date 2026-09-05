export const GITHUB_INSTALL_REQUEST_TITLE = "Waiting for approval";

export const GITHUB_INSTALL_REQUEST_NEXT = "After they approve, click Connect GitHub again.";

export const GITHUB_INSTALL_APPROVED_TITLE = "Request approved";

export const GITHUB_INSTALL_APPROVED_BODY =
  "The SuperPlane GitHub App is approved. The person who asked can click Connect GitHub again.";

export const GITHUB_INSTALL_APPROVED_ACTION = "Open SuperPlane";

export function githubInstallRequestBody(organization = ""): string {
  if (organization !== "") {
    return `Ask an admin of ${organization} to approve the SuperPlane GitHub App.`;
  }

  return "Ask a GitHub organization admin to approve the SuperPlane GitHub App.";
}

export function githubInstallRequestSettingsTitle(organization = ""): string {
  if (organization !== "") {
    return `Waiting for ${organization} approval`;
  }

  return "Waiting for GitHub approval";
}
