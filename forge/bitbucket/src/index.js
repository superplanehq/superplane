import { invokeRemote } from "@forge/api";

// bootstrap asks Forge to call SuperPlane so a lost system token can be
// delivered again. Open the web trigger URL when the cached token is gone.
export async function bootstrap() {
  const result = await invokeRemote("superplane", {
    path: "/api/v1/bitbucket/forge/bootstrap",
    method: "POST",
  });
  return {
    statusCode: result.status,
    body: "ok",
  };
}
