# PRODUCTION-BE onboarding investigation

## Status

Investigation date: 2026-10-29. The production failure is not reproduced.
The failing transition and node owner remain unknown. A repair needs more
evidence. This report does not establish that the incident is resolved.

## Confirmed evidence

- [Sentry event](https://superplane.sentry.io/issues/7815629869/events/ffcc81a0072b41e3b2a845977b663e44/):
  `ffcc81a0072b41e3b2a845977b663e44`, issue `7815629869`, short ID
  `PRODUCTION-BE`.
- The supplied event reports a `NotFoundError` from React DOM removal at
  `/onboarding`, at `2026-10-27T23:04:55.034000Z`.
- The supplied browser context is Android 10, Chrome Mobile 156.0.0, and
  React 19.2.8. The stack contains only React internals.
- The task records reporter confirmation of account connection with browser
  translation enabled.
- The supplied breadcrumbs contain successful GitHub and Bitbucket onboarding
  requests. They do not identify the connected service or response contents.

The service, last button, preceding actions, resulting screen, connection
state, and translation languages are unknown. No translated DOM capture is
available. No production-matched action sequence can be recorded yet.

## Evidence access

The available tool interface does not expose the named Sentry MCP server.
The event web URL returned access guidance instead of event details.
An unauthenticated Sentry event API request returned HTTP 401.
No Sentry environment keys were present.

`gh issue list --repo superplanehq/superplane --state all --search
'"PRODUCTION-BE"'` returned no matching issues. This search does not establish
that other incident discussion is absent.

The runner provides Playwright, but `playwright cli list` reported no browser
sessions. No native Android Chrome session or reporter session was available.
Native translation and its translation-disabled control were not exercised.

## Release comparison

The inspected implementation baseline is
`11b757e27773625265700f92852796dbb4b72013`.
The reported release is `b98d42e8c6f8ef7445af9bf333d472bf4bdbc21a`.
Both referenced comparison commits were fetched before comparing file contents.

The release-to-`8ac10718c4be33aad464719eba537d790b95698d` comparison shows no
changes in `App.tsx`, auth pages, onboarding, shared UI components, or art assets.

The release-to-baseline comparison does include onboarding changes, mainly in
model access, Codex subscription support, billing, and their tests.
`FirstRunSetup.tsx` changes affect agent and completion rendering.
`App.tsx` adds an admin statistics route.

The following inspected connection and ownership paths have identical contents
at the reported release and baseline:

- `OrganizationOnboardingRedirect.tsx`, `InitialWorkspaceOnboarding.tsx`, and
  `OnboardingPage.tsx`.
- `FirstRunConnectScreen.tsx`, `FirstRunBitbucketForgeScreen.tsx`,
  `FirstRunShell.tsx`, and `FirstRunSignedInAsLine.tsx`.
- `useFirstRunSetupFlow.ts`, `useFirstRunGitHub.ts`, `useFirstRunBitbucket.ts`,
  and `useBitbucketOnboarding.ts`.
- `FirstRunArtPane.tsx`, `onboardingArt.ts`, and public onboarding art assets.
- `web_src/src/components/ui/loading-button.tsx` and
  `web_src/src/ui/dropdownMenu/index.tsx`.

Onboarding file names above are relative to
`web_src/src/pages/factories/pages/onboarding/`, with visual components under
`first-run/`. The redirect is under `web_src/src/pages/auth/`.

## Transition and ownership findings

`App.tsx` renders `OrganizationOnboardingRedirect` at `/onboarding`.
The redirect provisions a workspace and renders `InitialWorkspaceOnboarding`.
Workspace, factory, and permission loading can replace the wizard with loading
or error content. `OnboardingPage` keys its workspace page by factory ID.
`FirstRunSetup` selects the visible first-run screen.

For GitHub, the connect screen replaces account-loading text with the stepper.
Its connection action sets a blocking state before browser navigation.
A linked identity can move the flow from connect to repository choice.
Back navigation explicitly prevents immediate forwarding to repository choice.

For Bitbucket, one screen selects loading, lookup failure, connect, grant,
or repository choice. Provider data refreshes independently of the selected host.
`useFirstRunSetupFlow` calls both provider hooks, so a Bitbucket fetch does not
prove that the user selected Bitbucket. Cached Bitbucket data remains available
during a refresh; pending and lookup-failure screens require missing data.

`LoadingButton` changes between children and a fragment containing a spinner
and text. The text has no element boundary. Translation could affect this
boundary, but no observed mutation implicates it in PRODUCTION-BE.

Account and organization menus use Radix portals. The inspected dropdown wrapper
does not manually remove portal nodes. No captured transition implicates a portal.

`FirstRunArtPane` passes an empty React-owned div to the renderer.
React-owned badges and pills are siblings of that div.
The external renderer creates children inside its target and clears that target
on mount and destroy. Destroy also cancels animation frames and disconnects its
resize observer. Source inspection does not show cleanup removing React-owned
children. This finding is not real-browser verification of the art lifecycle.

## Local checks

The following existing suites passed through the repository test target:

```sh
make check.test.ui FILES="src/pages/factories/pages/onboarding/FirstRunSetup.bitbucket.spec.tsx src/pages/factories/pages/onboarding/first-run/FirstRunBitbucketForgeScreen.spec.tsx src/pages/factories/pages/onboarding/first-run/FirstRunConnectScreen.spec.tsx src/pages/factories/pages/onboarding/first-run/FirstRunArtPane.spec.tsx src/pages/factories/pages/onboarding/first-run/FirstRunSignedInAsLine.spec.tsx"
```

The command reported 27 passing tests across five files, with no failures.
Coverage includes status refresh, Back navigation, repository choices, and a
delayed account switch. These checks did not reproduce the reported DOM error.
Happy DOM does not load the external art scripts; the art test uses a renderer mock.

`make format.js`, `make check.lint.ui`, and `make check.build.ui` passed.
The build reported dynamic-import and chunk-size warnings.
`git diff --check` passed for this documentation change.

## Evidence required to continue

Ask the reporter for these details:

1. Identify the service and whether the action connected, switched, or granted account access.
2. Record the preceding screen, last button, and screen visible after the error.
3. Record the active account, connection status, workspace choice, and repository choice using redacted identifiers.
4. Record the browser language, page language, translation target, and when translation started.
5. Record whether navigation returned from another tab, a popup, or an authentication redirect.
6. Repeat the same sequence with native translation enabled and disabled.

Obtain authenticated Sentry event details and any available replay or component
stack. Capture the translated DOM immediately before the failing update.
Identify the moved or replaced React-owned node and its actual parent.

Only then select the implicated owner and extend its existing regression fixture.
Show the same DOM removal failure before the repair and interactive continuation
after it. Verify connection status, repository selection, and reverse navigation.
Include delayed or repeated responses if the captured sequence requires them.

A generic DOM mutation simulation cannot establish a match to this incident.
The missing action sequence and native translated DOM evidence block a verified
repair. Production code and error reporting receive no changes in this report.
