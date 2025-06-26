# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v2.1.8](https://github.com/ljharb/Object.getOwnPropertyDescriptors/compare/v2.1.7...v2.1.8) - 2024-03-19

### Commits

- [actions] remove redundant finisher [`72e367f`](https://github.com/ljharb/Object.getOwnPropertyDescriptors/commit/72e367f708417284e465acb722ccaa0cff615a32)
- [Refactor] use `es-object-atoms` where possible [`a403fd2`](https://github.com/ljharb/Object.getOwnPropertyDescriptors/commit/a403fd2d4651586309b3a31a31010ca74fa5aa47)
- [Deps] update `call-bind`, `define-properties`, `es-abstract`, `safe-array-concat` [`fe4f40d`](https://github.com/ljharb/Object.getOwnPropertyDescriptors/commit/fe4f40d06ad1b877b1dd5aecb2e61c6d1c155c21)
- [Refactor] use `gopd` [`a2d1344`](https://github.com/ljharb/Object.getOwnPropertyDescriptors/commit/a2d1344271aa98bd9aa5771ea8847605628bcf6e)
- [Dev Deps] update `aud`, `mock-property`, `npmignore`, `tape` [`1852646`](https://github.com/ljharb/Object.getOwnPropertyDescriptors/commit/18526463da3df02dbc688e6897ab262245f85d99)

## [v2.1.7](https://github.com/ljharb/Object.getOwnPropertyDescriptors/compare/v2.1.6...v2.1.7) - 2023-08-31

### Commits

- [Deps] update `array.prototype.reduce`, `es-abstract` [`a201ad2`](https://github.com/ljharb/Object.getOwnPropertyDescriptors/commit/a201ad21768bbd199c122fc8b51ded6667b07573)
- [Dev Deps] update `@es-shims/api`, `@ljharb/eslint-config`, `aud`, `tape` [`f174ab1`](https://github.com/ljharb/Object.getOwnPropertyDescriptors/commit/f174ab14b2a816abac871d10747b1fcff179f0d8)

## [v2.1.6](https://github.com/ljharb/Object.getOwnPropertyDescriptors/compare/v2.1.5...v2.1.6) - 2023-04-20

### Commits

- [Refactor] use `safe-array-concat` [`a717eb2`](https://github.com/ljharb/Object.getOwnPropertyDescriptors/commit/a717eb21bacdc01eaa17092fe93dd21e0c1ef320)
- [Dev Deps] update `@es-shims/api`, `@ljharb/eslint-config`, `aud`, `tape` [`b08d70f`](https://github.com/ljharb/Object.getOwnPropertyDescriptors/commit/b08d70f9387aad6341d44d9216ffa36023973a66)
- [Deps] update `define-properties`, `es-abstract` [`95c8479`](https://github.com/ljharb/Object.getOwnPropertyDescriptors/commit/95c84794c2a78bc78c8c5f540db150c30fe9aea2)

<!-- auto-changelog-above -->

2.1.4 / 2022-05-19
=================
  * [Fix] `Array.prototype.reduce` isn’t present in ES3 engines
  * [meta] use `npmignore` to autogenerate an npmignore file
  * [Deps] update `define-properties`, `es-abstract`
  * [Dev Deps] update `eslint`, `@ljharb/eslint-config`, `@es-shims/api`, `aud`, `auto-changelog`, `functions-have-names`, `safe-publish-latest`, `tape`
  * [Tests] use `mock-property`
  * [actions] reuse common workflows
  * [actions] update codecov uploader

2.1.3 / 2021-10-03
=================
  * [readme] remove travis badge; add actions and codecov badges
  * [Deps] update `es-abstract`
  * [meta] use `prepublishOnly` script for npm 7+
  * [Dev Deps] update `eslint`, `@ljharb/eslint-config`, `@es-shims/api`, `aud`, `tape`
  * [actions] update workflows
  * [actions] use `node/install` instead of `node/run`; use `codecov` action

2.1.2 / 2021-02-20
=================
  * [Deps] update `call-bind`, `es-abstract`
  * [Dev Deps] update `eslint`, `@ljharb/eslint-config`, `aud`, `functions-have-names`, `has-strict-mode`, `tape`
  * [meta] do not publish github action workflow files
  * [meta] gitignore coverage output
  * [actions] update workflows

2.1.1 / 2020-11-26
=================
  * [Fix] do not mutate the native function when present
  * [Deps] update `es-abstract`; use `call-bind` where applicable
  * [meta] remove unused Makefile and associated utilities
  * [Dev Deps] update `eslint`, `@ljharb/eslint-config`, `tape`, `functions-have-names`; add `aud`
  * [actions] add Require Allow Edits workflow
  * [actions] switch Automatic Rebase workflow to `pull_request_target` event
  * [Tests] migrate tests to Github Actions
  * [Tests] run `nyc` on all tests
  * [Tests] add `implementation` test; run `es-shim-api` in postlint; use `tape` runner
  * [Tests] only audit prod deps

2.1.0 / 2019-12-12
=================
  * [New] add auto entry point
  * [Refactor] use split-up `es-abstract` (78% bundle size decrease)
  * [readme] fix repo URLs, remove testling
  * [Docs] Fix formatting in the README (#30)
  * [Deps] update `define-properties`, `es-abstract`
  * [Dev Deps] update `eslint`, `@ljharb/eslint-config`, `functions-have-names`, `covert`, `replace`, `semver`, `tape`, `@es-shims/api`; add `safe-publish-latest`
  * [meta] add `funding` field
  * [meta] Only apps should have lockfiles.
  * [Tests] use shared travis-ci configs
  * [Tests] use `functions-have-names`
  * [Tests] use `npx aud` instead of `nsp` or `npm audit` with hoops
  * [Tests] remove `jscs`
  * [actions] add automatic rebasing / merge commit blocking

2.0.3 / 2016-07-26
=================
  * [Fix] Update implementation to not return `undefined` descriptors
  * [Deps] update `es-abstract`
  * [Dev Deps] update `eslint`, `@ljharb/eslint-config`, `@es-shims/api`, `jscs`, `nsp`, `tape`, `semver`
  * [Dev Deps] remove unused eccheck script + dep
  * [Tests] up to `node` `v6.3`, `v5.12`, `v4.4`
  * [Tests] use pretest/posttest for linting/security
  * Update to stage 4

2.0.2 / 2016-01-27
=================
  * [Fix] ensure that `Object.getOwnPropertyDescriptors` does not fail when `Object.prototype` has a poisoned setter (#1, #2)

2.0.1 / 2016-01-27
=================
  * [Deps] move `@es-shims/api` to dev deps

2.0.0 / 2016-01-27
=================
  * [Breaking] implement the es-shims API
  * [Deps] update `define-properties`, `es-abstract`
  * [Dev Deps] update `tape`, `jscs`, `nsp`, `eslint`, `@ljharb/eslint-config`, `semver`
  * [Tests] fix npm upgrades in older nodes
  * [Tests] up to `node` `v5.5`
  * [Docs] Switch from vb.teelaun.ch to versionbadg.es for the npm version badge SVG

1.0.4 / 2015-07-20
=================
  * [Tests] Test on `io.js` `v2.4`
  * [Deps, Dev Deps] Update `define-properties`, `tape`, `eslint`, `semver`

1.0.3 / 2015-06-28
=================
  * Increase robustness by caching `Array#{concat, reduce}`
  * [Deps] Update `define_properties`
  * [Dev Deps] Update `eslint`, `semver`, `nsp`
  * [Tests] Test up to `io.js` `v2.3`

1.0.2 / 2015-05-23
=================
  * Update `es-abstract`, `tape`, `eslint`, `jscs`, `semver`, `covert`
  * Test up to `io.js` `v2.0`

1.0.1 / 2015-03-20
=================
  * Update `es-abstract`, `editorconfig-tools`, `nsp`, `eslint`, `semver`, `replace`

1.0.0 / 2015-02-17
=================
  * v1.0.0
