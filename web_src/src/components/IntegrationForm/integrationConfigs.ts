import type { IntegrationConfig } from "./types";

export const githubConfig: IntegrationConfig = {
  displayName: "GitHub",
  urlPlaceholder: "Johndoe",
  orgUrlLabel: "GitHub organization/owner name",
  validateUrl: (url: string): string | undefined => {
    if (!url.trim()) {
      return "Field cannot be empty";
    }

    try {
      const urlObj = new URL(url);
      if (!urlObj.origin.startsWith("https://github.com") || !urlObj.pathname.replace(/\/$/, "")) {
        return "Please provide a valid link to your GitHub organization or profile";
      }
      return undefined;
    } catch {
      return "Please provide a valid link to your GitHub organization or profile";
    }
  },
  extractOrgName: (url: string): string => {
    try {
      const urlObj = new URL(url);
      const pathParts = urlObj.pathname.split("/").filter(Boolean);
      return pathParts[0] || "";
    } catch {
      return "";
    }
  },
};

export const semaphoreConfig: IntegrationConfig = {
  displayName: "Semaphore",
  urlPlaceholder: "https://your-org.semaphoreci.com",
  orgUrlLabel: "Semaphore Org URL",
  validateUrl: (url: string): string | undefined => {
    if (!url.trim()) {
      return "Field cannot be empty";
    }

    try {
      const urlObj = new URL(url);
      if (!(urlObj.protocol === "http:" || urlObj.protocol === "https:")) {
        return "Please provide a valid link to your Semaphore organization";
      }
      return undefined;
    } catch {
      return "Please provide a valid link to your Semaphore organization";
    }
  },
  extractOrgName: (url: string): string => {
    try {
      const urlObj = new URL(url);
      const subdomain = urlObj.hostname.split(".")[0];
      return subdomain || "";
    } catch {
      return "";
    }
  },
};

export function getIntegrationConfig(integrationType: string): IntegrationConfig {
  switch (integrationType) {
    case "github":
      return githubConfig;
    case "semaphore":
      return semaphoreConfig;
    default:
      return semaphoreConfig;
  }
}
