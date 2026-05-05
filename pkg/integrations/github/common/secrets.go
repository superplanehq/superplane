package common

/*
 * These are the legacy secrets
 */
const GitHubAppPEM = "pem"
const GitHubAppClientSecret = "clientSecret"
const GitHubAppWebhookSecret = "webhookSecret"

/*
 * TODO: why are we using label-like names here?
 *
 * Secrets for the integration:
 * - Personal Access Token (PAT)
 * - GitHub App private key (PEM)
 */
const SecretPAT = "Personal Access Token"
const SecretAppClientSecret = "GitHub App Client Secret"
const SecretAppWebhookSecret = "GitHub App Webhook Secret"
const SecretAppPEM = "GitHub App Private Key (PEM)"
