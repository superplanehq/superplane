# App Discovery and Lifecycle

## Overview

SuperPlane users need to find, organize, create, install, configure, and remove
Apps. This feature covers the home experience and the related lifecycle
surfaces as one coherent entry point into App ownership.

## Terminology

- **App:** A deployable unit containing a workflow Canvas, Console, memory,
  files, versions, and operational history.
- **Folder:** An organization-level grouping used to organize Apps.
- **Install:** Create an App from a supported repository definition.

## Requirements

### REQ-APP-001: Discover and organize Apps

**User story:** As a workflow builder, I want to browse Apps and group them in
folders, so that I can quickly find the automation I need.

**Acceptance criteria:**

- **AC-APP-001.1:** When the organization contains Apps, SuperPlane shall list
  the Apps the builder may read and show their folder grouping.
- **AC-APP-001.2:** When the organization has no visible Apps, SuperPlane shall
  show an empty state that reflects whether the user may create one.

### REQ-APP-002: Create an App

**User story:** As a workflow builder with create access, I want to create a
blank App, so that I can author a new workflow.

**Acceptance criteria:**

- **AC-APP-002.1:** When the builder creates a valid blank App, SuperPlane
  shall open the new App and make it discoverable from the organization home.
- **AC-APP-002.2:** When a user lacks the permissions needed to create an App
  in a selected folder, SuperPlane shall leave the organization unchanged and
  explain that creation is unavailable.

### REQ-APP-003: Install a repository-defined App

**User story:** As a workflow builder, I want to preview and install an App
definition, so that I can adopt a reusable workflow with confidence.

**Acceptance criteria:**

- **AC-APP-003.1:** When a supported App source is previewed, SuperPlane shall
  show enough validated metadata for the builder to decide whether to install.
- **AC-APP-003.2:** When an App definition or bundled Console is invalid,
  SuperPlane shall report the validation failure without creating a partial
  App.

### REQ-APP-004: Manage App identity and lifecycle

**User story:** As an App owner, I want to update App settings and delete an
App, so that the organization's catalog remains accurate.

**Acceptance criteria:**

- **AC-APP-004.1:** When an authorized owner saves valid App settings,
  SuperPlane shall display the updated identity on App and home surfaces.
- **AC-APP-004.2:** When an authorized owner confirms deletion, SuperPlane
  shall remove the App from normal discovery and prevent new workflow runs.

## Traceability

- **Product context:** [Apps overview](../../README.md#what-it-does)
- **API evidence:** [Canvas and App lifecycle service](../../protos/canvases.proto)
  and [folder service](../../protos/canvas_folders.proto)
- **UI evidence:** [home and App routes](../../web_src/src/App.tsx) and
  [new App page](../../web_src/src/pages/home/NewAppPage.tsx)
- **Behavior evidence:** [home and folder behavior](../../test/e2e/home_page_test.go)
- **Install evidence:** [App installation endpoint](../../pkg/public/app_install.go)
  and [bundled Console contract](../../docs/prd/console-and-widgets.md#bundling-a-console-with-an-installable-app)
- **Feature blueprint:** [Canvas Authoring and Versioning](../blueprints/features/canvas-authoring-and-versioning.feature.md)

## Open Questions

- Which sources and trust signals qualify an App for install or featured
  discovery?
- Is App deletion recoverable, and if so for how long?
