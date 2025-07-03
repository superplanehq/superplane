# [9.1.0](https://github.com/ZeeCoder/use-resize-observer/compare/v9.0.2...v9.1.0) (2022-11-22)

### Bug Fixes

- trigger release ([da154e1](https://github.com/ZeeCoder/use-resize-observer/commit/da154e1db4e1c2b21fef722d998fd2adaba46d9f))

### Features

- Exported some TS types ([3895bdf](https://github.com/ZeeCoder/use-resize-observer/commit/3895bdf98687f285e707697d451231ab7f46d189)), closes [#98](https://github.com/ZeeCoder/use-resize-observer/issues/98)

# [9.1.0-alpha.1](https://github.com/ZeeCoder/use-resize-observer/compare/v9.0.2...v9.1.0-alpha.1) (2022-11-21)

### Features

- Exported some TS types ([e056e19](https://github.com/ZeeCoder/use-resize-observer/commit/e056e191499c713dc7bdd872530cfb9786e1af3e)), closes [#98](https://github.com/ZeeCoder/use-resize-observer/issues/98)

## [9.0.2](https://github.com/ZeeCoder/use-resize-observer/compare/v9.0.1...v9.0.2) (2022-06-13)

### Bug Fixes

- trigger release ([01fccc6](https://github.com/ZeeCoder/use-resize-observer/commit/01fccc6bf5036903a650b42ded58fda4e2907149))

## [9.0.1](https://github.com/ZeeCoder/use-resize-observer/compare/v9.0.0...v9.0.1) (2022-06-13)

### Bug Fixes

- Made element resolution compatbile with concurrent mode ([c9c6689](https://github.com/ZeeCoder/use-resize-observer/commit/c9c66894abeb8fefd5916a0c119be809357fcdf5))
- Small changes and chores ([0cb6800](https://github.com/ZeeCoder/use-resize-observer/commit/0cb68008619976c45dfa133a332c75766a43d7d4))

# [9.0.0](https://github.com/ZeeCoder/use-resize-observer/compare/v8.0.0...v9.0.0) (2022-05-15)

### Bug Fixes

- Added some fixes for React 18. ([852d976](https://github.com/ZeeCoder/use-resize-observer/commit/852d976e481a215671be95964c6fa05825eee82a)), closes [#90](https://github.com/ZeeCoder/use-resize-observer/issues/90) [#91](https://github.com/ZeeCoder/use-resize-observer/issues/91) [#92](https://github.com/ZeeCoder/use-resize-observer/issues/92)

### BREAKING CHANGES

- The lib now takes "Element", not only "HTMLElement", to
  be consistent with ResizeObserver.

# [8.0.0](https://github.com/ZeeCoder/use-resize-observer/compare/v7.0.1...v8.0.0) (2021-08-28)

### Bug Fixes

- The `onResize` callback is no longer incorrectly called with the same values. ([bd0f3c8](https://github.com/ZeeCoder/use-resize-observer/commit/bd0f3c8597bac0d853b88cf585256aac1bd4f554))

### Features

- Added the `box` option ([0ca6c23](https://github.com/ZeeCoder/use-resize-observer/commit/0ca6c23dd5573526f1dd716851083f922ca73f68)), closes [#31](https://github.com/ZeeCoder/use-resize-observer/issues/31) [#57](https://github.com/ZeeCoder/use-resize-observer/issues/57)
- Added the `round` option. ([aa38199](https://github.com/ZeeCoder/use-resize-observer/commit/aa38199f21f60bd4a361a2198e9e5f200bf5287c)), closes [#55](https://github.com/ZeeCoder/use-resize-observer/issues/55) [#46](https://github.com/ZeeCoder/use-resize-observer/issues/46) [#61](https://github.com/ZeeCoder/use-resize-observer/issues/61)

### BREAKING CHANGES

- Removed `resize-observer-polyfill` in favour of `@juggle/resize-observer`. ([8afc8f6](https://github.com/ZeeCoder/use-resize-observer/commit/8afc8f6c52ee047a41ac107379ebdf27e1a95997))

# [7.1.0](https://github.com/ZeeCoder/use-resize-observer/compare/v7.0.1...v7.1.0) (2021-08-28)

**This was an accidental release**, and an equivalent of V8.

## [7.0.1](https://github.com/ZeeCoder/use-resize-observer/compare/v7.0.0...v7.0.1) (2021-07-27)

### Bug Fixes

- Removed unnecessary entries.length check ([3211d33](https://github.com/ZeeCoder/use-resize-observer/commit/3211d338117b0d2a97ccb229683eb8458de81d01))
- Undefined HTMLElement is no longer an issue in certain SSR edge cases. ([599cace](https://github.com/ZeeCoder/use-resize-observer/commit/599cace5c33ecd4276a0fe2848e0ed920f81e2fe)), closes [#74](https://github.com/ZeeCoder/use-resize-observer/issues/74) [#62](https://github.com/ZeeCoder/use-resize-observer/issues/62)

# [7.0.0](https://github.com/ZeeCoder/use-resize-observer/compare/v6.1.0...v7.0.0) (2020-11-11)

### Bug Fixes

- Only instantiating a ResizeObserver instance if there's actually something to
  observe. This for example means that if you pass in `null` or undefined as the
  ref, or if neither the default ref or RefCallback returned from the hook are
  in use, then no ResizeObserver instance will get created until there's an
  actual element to observe. Resolves: #42
- Fixed an error where in certain edge cases the hook tried to set state when
  its host component already unmounted.

### Features

- The `ref` option now accepts raw elements as well.

### BREAKING CHANGES

- The returned ref is now a RefCallback, not a ref object. Resolves: #43, #45
- The returned ref will always be the same RefCallback.
  Previously when a custom ref object was passed, it was returned as well from
  the hook as "ref".
- Compared to 6.2.0-alpha.1 There's no `callbackRef` return value anymore.

### Misc

- Using package.json file attr instead of gitignore ([c58f34e](https://github.com/ZeeCoder/use-resize-observer/commit/c58f34e11b68ef9622a6b2528da8ee68a9685211))
- Added Semantic Release ([55f6368](https://github.com/ZeeCoder/use-resize-observer/commit/55f6368c1b0c3154bfd6ed16e089763de0b0ba47))
- Handling custom refs (through options), the default ref and the RefCallback
  has been greatly refactored internally (into the `useResolvedElement`
  hook), to handle more edge cases with the way refs are handled.
- Tests based on react testing library were refactored to make them much simpler
  and more approachable.
- Added [contributing guidelines](./CONTRIBUTING.md)
- Added tests in real browsers with BrowserStack, so that we ensure the lib
  works all the way back to IE11.
- Switched to GitHub Actions from Travis, as builds started to freeze. (They've
  also announced a [limit on OS projects](https://blog.travis-ci.com/2020-11-02-travis-ci-new-billing).)

# [7.0.0-alpha.4](https://github.com/ZeeCoder/use-resize-observer/compare/v7.0.0-alpha.3...v7.0.0-alpha.4) (2020-11-11)

### Bug Fixes

- Using package.json file attr instead of gitignore ([74ea0a9](https://github.com/ZeeCoder/use-resize-observer/commit/74ea0a97c3575388506536a700586aecf0ba0816))

# [7.0.0-alpha.3](https://github.com/ZeeCoder/use-resize-observer/compare/v7.0.0-alpha.2...v7.0.0-alpha.3) (2020-11-11)

### Bug Fixes

- Added Semantic Release ([5074c0f](https://github.com/ZeeCoder/use-resize-observer/commit/5074c0fefd29e53a8ed9a4672ba043ad3be6d972), [54a83ce](https://github.com/ZeeCoder/use-resize-observer/commit/54a83cede6fcb8dbfa9e0f9a0ea2f1f4557b606f))

# [7.0.0-alpha.2](https://github.com/ZeeCoder/use-resize-observer/compare/v7.0.0-alpha.1...v7.0.0-alpha.2) (2020-11-11)

Skipped Release
