#!/usr/bin/env node

/* eslint-disable no-console */

const crypto = require("node:crypto");
const fs = require("node:fs");
const http = require("node:http");
const https = require("node:https");

function base64url(value) {
  return Buffer.from(value).toString("base64url");
}

function createJWT(appID, privateKey, now = Math.floor(Date.now() / 1000)) {
  const header = base64url(JSON.stringify({ alg: "RS256", typ: "JWT" }));
  const payload = base64url(
    JSON.stringify({ iat: now - 60, exp: now + 540, iss: appID }),
  );
  const unsigned = `${header}.${payload}`;
  const signature = crypto.sign(
    "RSA-SHA256",
    Buffer.from(unsigned),
    privateKey,
  );
  return `${unsigned}.${signature.toString("base64url")}`;
}

function requestToken({ apiURL, installationID, jwt }) {
  const url = new URL(
    `/app/installations/${encodeURIComponent(installationID)}/access_tokens`,
    apiURL,
  );
  const transport = url.protocol === "http:" ? http : https;
  return new Promise((resolve, reject) => {
    const request = transport.request(
      url,
      {
        method: "POST",
        headers: {
          Accept: "application/vnd.github+json",
          Authorization: `Bearer ${jwt}`,
          "User-Agent": "superplane-component-release",
          "X-GitHub-Api-Version": "2022-11-28",
          "Content-Length": "0",
        },
      },
      (response) => {
        const chunks = [];
        response.on("data", (chunk) => chunks.push(chunk));
        response.on("end", () => {
          const body = Buffer.concat(chunks).toString("utf8");
          if (response.statusCode !== 201) {
            reject(
              new Error(
                `create GitHub App installation token failed with HTTP ${response.statusCode}: ${body}`,
              ),
            );
            return;
          }
          try {
            const token = JSON.parse(body).token;
            if (!token) {
              reject(
                new Error(
                  "GitHub App installation token response has no token",
                ),
              );
              return;
            }
            resolve(token);
          } catch (error) {
            reject(
              new Error(
                `GitHub App token response is invalid JSON: ${error.message}`,
              ),
            );
          }
        });
      },
    );
    request.on("error", reject);
    request.end();
  });
}

async function createInstallationToken({
  apiURL,
  appID,
  installationID,
  privateKey,
}) {
  if (!appID || !installationID || !privateKey) {
    throw new Error(
      "GitHub App ID, installation ID, and private key are required",
    );
  }
  return requestToken({
    apiURL,
    installationID,
    jwt: createJWT(appID, privateKey),
  });
}

async function main() {
  const keyPath =
    process.env.GITHUB_RELEASE_APP_PRIVATE_KEY_PATH ||
    "/home/semaphore/github-release-app.pem";
  const token = await createInstallationToken({
    apiURL: process.env.GITHUB_API_URL || "https://api.github.com",
    appID: process.env.GITHUB_RELEASE_APP_ID,
    installationID: process.env.GITHUB_RELEASE_APP_INSTALLATION_ID,
    privateKey: fs.readFileSync(keyPath, "utf8"),
  });
  process.stdout.write(`${token}\n`);
}

if (require.main === module) {
  main().catch((error) => {
    console.error(`Error: ${error.message}`);
    process.exit(1);
  });
}

module.exports = { createInstallationToken, createJWT };
