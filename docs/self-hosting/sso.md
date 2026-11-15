# Single Sign-On with Okta

Connect one Okta organization to a SuperPlane organization. Members then sign
in through Okta. Okta can also create and deactivate members.

You need:

- A SuperPlane Enterprise license with the `sso` feature granted to the
  organization.
- A public HTTPS base URL.
- A DNS domain that you control.
- An Okta developer organization.
- Okta users whose email addresses use that domain.

## 1. Verify the domain

1. Open **Settings** and select **Single Sign-On**.
2. Add the domain.
3. Create a DNS TXT record. The name is
   `_superplane-verification.<domain>`. The value is
   `superplane-domain-verification=<token>`. Copy both values from the page.
4. Select **Verify**. The status becomes **Verified**.

## 2. Create the Okta application

1. In Okta, create an **OIDC - Web Application**.
2. Copy these values from the SuperPlane page into Okta:
   - Sign-in redirect URI
   - Initiate login URI
   - Sign-out redirect URI
   - Back-channel logout URI
3. Assign the test users.

## 3. Save and activate the connection

1. In SuperPlane, select the **Okta** preset.
2. Enter the issuer `https://<okta-domain>`, the client ID, and the client
   secret.
3. Save the connection.
4. Select **Test connection**. The page shows the subject, email, and name
   from the ID token.
5. Select **Activate**.

`GET /auth/config` then returns `ssoEnabled: true`.

## 4. Sign in

Open a private window. Select **Continue with SSO**, enter the email, and
complete the Okta sign-in. SuperPlane opens the organization. The member has
the default role and the badge **SSO**.

These URLs also sign the member in:

- `/login/sso/<organization-slug>`
- The Okta dashboard tile, which uses the initiate login URI

## 5. Require Single Sign-On

Set enforcement to **Required**. A member with a non-SSO session is sent to
Okta. An owner can still use another method when owner bypass is on.

## 6. Provision members with SCIM

1. In Okta, add the **SCIM 2.0 Test App (OAuth Bearer Token)** application.
2. Copy the SCIM base URL and a SCIM token from the SuperPlane page.
3. Enable create, update, and deactivate.
4. Assign a user. The member appears with the badge **SCIM** before the first
   sign-in.
5. Unassign the user. SuperPlane removes the member. The next request from
   that user does not open the organization.

Okta sends `GET /Users?filter=userName eq "..."`, then `POST /Users`. Profile
updates use `PUT`. Deactivate and reactivate use `PATCH` with `active`. Okta
does not send `DELETE`.

## 7. Sign out

Turn on **Sign out from the identity provider**. Sign out of SuperPlane. The
Okta session ends too.

End the user session in the Okta admin console. SuperPlane receives a
back-channel logout and the session loses access to the organization. If the
ID token has no `sid`, SuperPlane matches the logout by the subject.

## 8. Installation administrator

An installation administrator can open the organization and select **Disable
connection** or **Verify domain manually**.

## Rate limit

`POST /auth/sso/discover` allows 10 requests per minute for each client
address. SuperPlane reads the address from the connection, not from
`X-Forwarded-For`. Behind a reverse proxy, the limit applies to the proxy
address.
