import * as superplane from "@superplane/sdk";
import * as crypto from "crypto";

const PEM_SECRET = "pem";
const CLIENT_SECRET = "clientSecret";
const WEBHOOK_SECRET = "webhookSecret";

interface AppMetadata {
  installationId?: string;
  state?: string;
  owner?: string;
  repositories?: Array<{ id: number; name: string; url: string }>;
  githubApp?: { id: number; slug: string; clientId: string };
}

// --- GitHub App JWT auth ---

function createJWT(appId: number, pem: string): string {
  const header = Buffer.from(
    JSON.stringify({ alg: "RS256", typ: "JWT" })
  ).toString("base64url");

  const now = Math.floor(Date.now() / 1000);
  const payload = Buffer.from(
    JSON.stringify({ iat: now - 60, exp: now + 600, iss: appId })
  ).toString("base64url");

  const signature = crypto
    .createSign("RSA-SHA256")
    .update(`${header}.${payload}`)
    .sign(pem, "base64url");

  return `${header}.${payload}.${signature}`;
}

async function getInstallationToken(
  http: superplane.HTTPClient,
  appId: number,
  installationId: string,
  pem: string
): Promise<string> {
  const jwt = createJWT(appId, pem);
  const resp = await http.request(
    "POST",
    `https://api.github.com/app/installations/${installationId}/access_tokens`,
    {
      headers: {
        Authorization: `Bearer ${jwt}`,
        Accept: "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
      },
    }
  );

  if (resp.status !== 201) {
    throw new Error(
      `Failed to get installation token: ${resp.status} ${resp.body?.message}`
    );
  }

  return resp.body.token;
}

function findSecret(
  secrets: Array<{ name: string; value: string }>,
  name: string
): string {
  const s = secrets.find((s) => s.name === name);
  if (!s) throw new Error(`Secret '${name}' not found`);
  return s.value;
}

// --- Plugin activation ---

export function activate(context: superplane.PluginContext) {
  // Register the integration handler for the GitHub App lifecycle
  context.integrations.register("github-issues", {
    async sync(ctx) {
      const metadata: AppMetadata =
        (await ctx.integration.getMetadata()) || {};

      if (metadata.installationId) {
        return;
      }

      const state = crypto.randomBytes(32).toString("base64url");
      const integrationId = await ctx.integration.id();

      const config = ctx.configuration || {};
      const org = config.organization || "";

      const manifest = JSON.stringify({
        name: "SuperPlane GH integration",
        public: false,
        url: "https://superplane.com",
        default_permissions: {
          issues: "write",
          repository_hooks: "write",
        },
        default_events: ["issues"],
        setup_url: `${ctx.baseUrl}/api/v1/integrations/${integrationId}/setup`,
        redirect_url: `${ctx.baseUrl}/api/v1/integrations/${integrationId}/redirect`,
        hook_attributes: {
          url: `${ctx.webhooksBaseUrl}/api/v1/integrations/${integrationId}/webhook`,
        },
      });

      const browserUrl = org
        ? `https://github.com/organizations/${org}/settings/apps/new`
        : "https://github.com/settings/apps/new";

      await ctx.integration.newBrowserAction({
        description: `To complete the GitHub app setup:\n\n1. The "**Continue**" button/link will take you to GitHub with the app manifest pre-filled.\n2. **Create GitHub App**: Give the new app a name, and click the "Create" button.\n3. **Install GitHub App**: Install the new GitHub app in the user/organization.`,
        url: browserUrl,
        method: "POST",
        formFields: { manifest, state },
      });

      await ctx.integration.setMetadata({ owner: org, state });
    },

    async handleRequest(ctx) {
      const metadata: AppMetadata =
        (await ctx.integration.getMetadata()) || {};

      if (ctx.request.path.endsWith("/redirect")) {
        return handleAfterAppCreation(ctx, metadata);
      }

      if (ctx.request.path.endsWith("/setup")) {
        return handleAfterAppInstallation(ctx, metadata);
      }

      if (ctx.request.path.endsWith("/webhook")) {
        return handleIntegrationWebhook(ctx, metadata);
      }

      return { action: "error", status: 404, message: "not found" };
    },

    webhookHandler: {
      async setup(ctx) {
        const secrets = await ctx.integration.getSecrets();
        const pem = findSecret(secrets, PEM_SECRET);
        const metadata: AppMetadata =
          (await ctx.integration.getMetadata()) || {};

        if (!metadata.githubApp || !metadata.installationId) {
          throw new Error("GitHub App not configured");
        }

        const token = await getInstallationToken(
          ctx.http,
          metadata.githubApp.id,
          metadata.installationId,
          pem
        );

        const config = ctx.configuration as {
          eventType?: string;
          repository?: string;
        };

        const resp = await ctx.http.request(
          "POST",
          `https://api.github.com/repos/${metadata.owner}/${config.repository}/hooks`,
          {
            headers: {
              Authorization: `Bearer ${token}`,
              Accept: "application/vnd.github+json",
              "Content-Type": "application/json",
              "X-GitHub-Api-Version": "2022-11-28",
            },
            body: JSON.stringify({
              active: true,
              events: [config.eventType || "issues"],
              config: {
                url: ctx.webhookUrl,
                secret: ctx.webhookSecret,
                content_type: "json",
              },
            }),
          }
        );

        if (resp.status !== 201) {
          throw new Error(
            `Failed to create webhook: ${resp.status} ${resp.body?.message}`
          );
        }

        return { id: resp.body.id, name: resp.body.name };
      },

      async cleanup(ctx) {
        const secrets = await ctx.integration.getSecrets();
        const pem = findSecret(secrets, PEM_SECRET);
        const metadata: AppMetadata =
          (await ctx.integration.getMetadata()) || {};

        if (!metadata.githubApp || !metadata.installationId) {
          return;
        }

        const token = await getInstallationToken(
          ctx.http,
          metadata.githubApp.id,
          metadata.installationId,
          pem
        );

        const config = ctx.configuration as { repository?: string };
        const webhookMeta = ctx.webhookMetadata as { id?: number };

        if (!webhookMeta?.id) return;

        await ctx.http.request(
          "DELETE",
          `https://api.github.com/repos/${metadata.owner}/${config.repository}/hooks/${webhookMeta.id}`,
          {
            headers: {
              Authorization: `Bearer ${token}`,
              Accept: "application/vnd.github+json",
              "X-GitHub-Api-Version": "2022-11-28",
            },
          }
        );
      },

      async compareConfig(a: any, b: any) {
        return a?.repository === b?.repository && a?.eventType === b?.eventType;
      },
    },
  });

  // Register the create-issue component
  context.components.register("github-issues.create-issue", {
    async setup(ctx) {
      // Validation only -- ensure repo is accessible
    },

    async execute(ctx) {
      const secrets = await ctx.integration.getSecrets();
      const pem = findSecret(secrets, PEM_SECRET);
      const metadata: AppMetadata =
        (await ctx.integration.getMetadata()) || {};

      if (!metadata.githubApp || !metadata.installationId) {
        ctx.fail("not_configured", "GitHub App is not configured");
        return;
      }

      const token = await getInstallationToken(
        ctx.http,
        metadata.githubApp.id,
        metadata.installationId,
        pem
      );

      const config = ctx.configuration;
      const resp = await ctx.http.request(
        "POST",
        `https://api.github.com/repos/${metadata.owner}/${config.repository}/issues`,
        {
          headers: {
            Authorization: `Bearer ${token}`,
            Accept: "application/vnd.github+json",
            "Content-Type": "application/json",
            "X-GitHub-Api-Version": "2022-11-28",
          },
          body: JSON.stringify({
            title: config.title,
            body: config.body || "",
            assignees: config.assignees || [],
            labels: config.labels || [],
          }),
        }
      );

      if (resp.status !== 201) {
        ctx.fail(
          "api_error",
          `GitHub API returned ${resp.status}: ${resp.body?.message}`
        );
        return;
      }

      ctx.emit("default", "github.issue", resp.body);
    },
  });

  // Register the on-issue-created trigger
  context.triggers.register("github-issues.on-issue-created", {
    async setup(ctx) {
      const config = ctx.configuration as { repository?: string };
      if (!config.repository) {
        throw new Error("repository is required");
      }

      await ctx.integration.requestWebhook({
        eventType: "issues",
        repository: config.repository,
      });
    },

    async handleWebhook(ctx) {
      const signature = ctx.headers["x-hub-signature-256"];
      const secret = await ctx.webhook.getSecret();

      const expected =
        "sha256=" +
        crypto.createHmac("sha256", secret).update(ctx.body).digest("hex");

      if (signature !== expected) {
        return { status: 401 };
      }

      const payload = JSON.parse(ctx.body);

      if (payload.action !== "opened") {
        return { status: 200 };
      }

      ctx.events.emit("github.issue", payload);
      return { status: 200 };
    },
  });
}

export function deactivate() {}

// --- Integration request handlers ---

async function handleAfterAppCreation(
  ctx: superplane.IntegrationRequestContext,
  metadata: AppMetadata
): Promise<superplane.RequestResponse> {
  const code = ctx.request.query.code;
  const state = ctx.request.query.state;

  if (!code || !state || state !== metadata.state) {
    return { action: "error", status: 400, message: "missing code or state" };
  }

  const resp = await ctx.http.request(
    "POST",
    `https://api.github.com/app-manifests/${code}/conversions`,
    {
      headers: {
        Accept: "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
      },
    }
  );

  if (resp.status !== 201) {
    return {
      action: "error",
      status: 500,
      message: `Failed to convert manifest: ${resp.body?.message}`,
    };
  }

  const appData = resp.body;

  metadata.githubApp = {
    id: appData.id,
    slug: appData.slug,
    clientId: appData.client_id,
  };

  await ctx.integration.setMetadata(metadata);
  await ctx.integration.setSecret(CLIENT_SECRET, appData.client_secret);
  await ctx.integration.setSecret(WEBHOOK_SECRET, appData.webhook_secret);
  await ctx.integration.setSecret(PEM_SECRET, appData.pem);

  return {
    action: "redirect",
    url: `https://github.com/apps/${appData.slug}/installations/new?state=${state}`,
  };
}

async function handleAfterAppInstallation(
  ctx: superplane.IntegrationRequestContext,
  metadata: AppMetadata
): Promise<superplane.RequestResponse> {
  const integrationId = await ctx.integration.id();

  if (metadata.installationId) {
    return {
      action: "redirect",
      url: `${ctx.baseUrl}/${ctx.organizationId}/settings/integrations/${integrationId}`,
    };
  }

  const installationId = ctx.request.query.installation_id;
  const setupAction = ctx.request.query.setup_action;
  const state = ctx.request.query.state;

  if (!installationId || state !== metadata.state) {
    return {
      action: "error",
      status: 400,
      message: "invalid installation ID or state",
    };
  }

  if (setupAction !== "install") {
    return { action: "json", status: 200, body: { ok: true } };
  }

  metadata.installationId = installationId;

  const secrets = await ctx.integration.getSecrets();
  const pem = findSecret(secrets, PEM_SECRET);
  const token = await getInstallationToken(
    ctx.http,
    metadata.githubApp!.id,
    installationId,
    pem
  );

  if (!metadata.owner) {
    const appResp = await ctx.http.request(
      "GET",
      `https://api.github.com/apps/${metadata.githubApp!.slug}`,
      {
        headers: {
          Authorization: `Bearer ${createJWT(metadata.githubApp!.id, pem)}`,
          Accept: "application/vnd.github+json",
          "X-GitHub-Api-Version": "2022-11-28",
        },
      }
    );
    if (appResp.status === 200 && appResp.body?.owner?.login) {
      metadata.owner = appResp.body.owner.login;
    }
  }

  const reposResp = await ctx.http.request(
    "GET",
    "https://api.github.com/installation/repositories",
    {
      headers: {
        Authorization: `Bearer ${token}`,
        Accept: "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
      },
    }
  );

  metadata.repositories = [];
  if (reposResp.status === 200 && reposResp.body?.repositories) {
    for (const r of reposResp.body.repositories) {
      metadata.repositories.push({
        id: r.id,
        name: r.name,
        url: r.html_url,
      });
    }
  }

  metadata.state = "";
  await ctx.integration.setMetadata(metadata);
  await ctx.integration.removeBrowserAction();
  await ctx.integration.ready();

  return {
    action: "redirect",
    url: `${ctx.baseUrl}/${ctx.organizationId}/settings/integrations/${integrationId}`,
  };
}

async function handleIntegrationWebhook(
  ctx: superplane.IntegrationRequestContext,
  metadata: AppMetadata
): Promise<superplane.RequestResponse> {
  const secrets = await ctx.integration.getSecrets();
  let webhookSecret: string;
  try {
    webhookSecret = findSecret(secrets, WEBHOOK_SECRET);
  } catch {
    return { action: "error", status: 500, message: "webhook secret not found" };
  }

  const signature = ctx.request.headers["X-Hub-Signature-256"] || ctx.request.headers["x-hub-signature-256"] || "";
  const expected =
    "sha256=" +
    crypto
      .createHmac("sha256", webhookSecret)
      .update(ctx.request.body)
      .digest("hex");

  if (signature !== expected) {
    return { action: "error", status: 400, message: "invalid signature" };
  }

  const event = JSON.parse(ctx.request.body);
  const eventType = ctx.request.headers["X-GitHub-Event"] || ctx.request.headers["x-github-event"] || "";

  if (eventType === "installation") {
    const action = event.action;
    if (action === "suspend") {
      await ctx.integration.error("app installation was suspended");
    } else if (action === "unsuspend") {
      await ctx.integration.ready();
    } else if (action === "deleted") {
      metadata.installationId = "";
      metadata.repositories = [];
      const state = crypto.randomBytes(32).toString("base64url");
      metadata.state = state;
      await ctx.integration.setMetadata(metadata);
      await ctx.integration.error("app installation was deleted");
      if (metadata.githubApp?.slug) {
        await ctx.integration.newBrowserAction({
          description: `To complete the GitHub app setup:\n1. **Install GitHub App**: Install the new GitHub app in the user/organization.`,
          url: `https://github.com/apps/${metadata.githubApp.slug}/installations/new?state=${state}`,
          method: "GET",
        });
      }
    }
  }

  return { action: "json", status: 200, body: { ok: true } };
}
