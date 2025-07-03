# CONTRIBUTING

When contributing to this project, please keep in mind the following:

- The hook must remain as simple as possible. It's only a low-level "proxy" to a
  ResizeObserver instance aiming for correctness, which should not add or polyfill
  features on top. All that can be done by composing hooks.
- All features must be covered with test(s).

It's also best to first submit an issue, before creating a pull request so that
the required feature can be discussed, as well as the actual implementation.

## Adding a New Feature

- Open an issue, so that a discussion can take place,
- Once discussed, add the changes to `src/index.ts` accordingly (make sure TS
  types are respected as well),
- Add new test(s) to cover the new feature in: `test/testing-lib.tsx`.
  Ignore the other test files, as they're fairly old and much harder to add new
  ones to them compared to just using react testing lib.
- Run all the tests to ensure there are no regressions: `yarn test`,

## Using Watch Modes While Developing

There are other test-related scripts in package.json, to enhance the developer
experience.
While making changes you might want to watch the source files, and build them
automatically, as well as having Karma run a (non-headless) Chrome instance
every time a change was made.

To do so:

- Run `yarn src:watch` in a terminal tab
- Run `yarn karma:watch` in another.

Don't forget to run `yarn test` at the end once you're done with everything, to
make sure the new code is tested for regressions.

If you have a Browserstack account, then you can also run the tests in real browsers using the `test:bs:*` commands.
Just make sure you have the following env variables set: `BS_USERNAME`, `BS_ACCESS_KEY`.
