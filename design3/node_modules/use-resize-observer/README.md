# use-resize-observer


<h1 align="center">
	<br>
	<img width="250" src="https://raw.githubusercontent.com/ZeeCoder/use-resize-observer/master/media/Logo.png" alt="useResizeObserver">
	<br>
    <br>
</h1>

A React hook that allows you to use a ResizeObserver to measure an element's size.

[![npm version](https://badge.fury.io/js/use-resize-observer.svg)](https://npmjs.com/package/use-resize-observer)
[![build](https://github.com/ZeeCoder/use-resize-observer/workflows/Testing/badge.svg)](https://github.com/ZeeCoder/use-resize-observer/actions/workflows/testing.yml)
[![BrowserStack Status](https://automate.browserstack.com/badge.svg?badge_key=aVpjV2RZbThnWnh2S0FvREh0cGRtRHRCNzYwUmw4N0Z4WUxybHM0WkpqST0tLW9RT0tDeGk3OVU2WkNtalpON29xWFE9PQ==--ec6a97c52cd7ad30417612ca3f5df511eef5d631)](https://automate.browserstack.com/public-build/aVpjV2RZbThnWnh2S0FvREh0cGRtRHRCNzYwUmw4N0Z4WUxybHM0WkpqST0tLW9RT0tDeGk3OVU2WkNtalpON29xWFE9PQ==--ec6a97c52cd7ad30417612ca3f5df511eef5d631)

## Highlights

- Written in **TypeScript**.
- **Tiny**: [648B](.size-limit.json) (minified, gzipped) Monitored by [size-limit](https://github.com/ai/size-limit).
- Exposes an **onResize callback** if you need more control.
- `box` [option](https://developer.mozilla.org/en-US/docs/Web/API/ResizeObserver/observe#syntax).
- Works with **SSR**.
- Works with **CSS-in-JS**.
- **Supports custom refs** in case you [had one already](#passing-in-your-own-ref).
- **Uses RefCallback by default** To address delayed mounts and changing ref elements.
- **Ships a polyfilled version**
- Handles many edge cases you might not even think of.
  (See this documentation and the test cases.)
- Easy to compose ([Throttle / Debounce](#throttle--debounce), [Breakpoints](#breakpoints))
- **Tested in real browsers** (Currently latest Chrome, Firefox, Edge, Safari, Opera, IE 11, iOS and Android, sponsored by BrowserStack)

## In Action

[CodeSandbox Demo](https://codesandbox.io/s/nrp0w2r5z0)

## Install

```sh
yarn add use-resize-observer --dev
# or
npm install use-resize-observer --save-dev
```

## Options

| Option   | Type                                                                                 | Description                                                                                                                   | Default        |
| -------- | ------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------- | -------------- |
| ref      | undefined &#124; RefObject &#124; HTMLElement                                        | A ref or element to observe.                                                                                                  | undefined      |
| box      | undefined &#124; "border-box" &#124; "content-box" &#124; "device-pixel-content-box" | The [box model](https://developer.mozilla.org/en-US/docs/Web/API/ResizeObserver/observe#syntax) to use for observation.       | "content-box"  |
| onResize | undefined &#124; ({ width?: number, height?: number }) => void                       | A callback receiving the element size. If given, then the hook will not return the size, and instead will call this callback. | undefined      |
| round    | undefined &#124; (n: number) => number                                               | A function to use for rounding values instead of the default.                                                                 | `Math.round()` |

## Response

| Name   | Type                    | Description                                    |
| ------ | ----------------------- | ---------------------------------------------- |
| ref    | RefCallback             | A callback to be passed to React's "ref" prop. |
| width  | undefined &#124; number | The width (or "blockSize") of the element.     |
| height | undefined &#124; number | The height (or "inlineSize") of the element.   |

## Basic Usage

Note that the default builds are not polyfilled! For instructions and alternatives,
see the [Transpilation / Polyfilling](#transpilation--polyfilling) section.

```tsx
import React from "react";
import useResizeObserver from "use-resize-observer";

const App = () => {
  const { ref, width = 1, height = 1 } = useResizeObserver<HTMLDivElement>();

  return (
    <div ref={ref}>
      Size: {width}x{height}
    </div>
  );
};
```

To observe a different box size other than content box, pass in the `box` option, like so:

```tsx
const { ref, width, height } = useResizeObserver<HTMLDivElement>({
  box: "border-box",
});
```

Note that if the browser does not support the given box type, then the hook won't report any sizes either.

### Box Options

Note that box options are experimental, and as such are not supported by all browsers that implemented ResizeObservers. (See [here](https://developer.mozilla.org/en-US/docs/Web/API/ResizeObserverEntry).)

`content-box` (default)

Safe to use by all browsers that implemented ResizeObservers. The hook internally will fall back to `contentRect` from
the old spec in case `contentBoxSize` is not available.

`border-box`

Supported well for the most part by evergreen browsers. If you need to support older versions of these browsers however,
then you may want to feature-detect for support, and optionally include a polyfill instead of the native implementation.

`device-pixel-content-box`

Surma has a [very good article](https://web.dev/device-pixel-content-box/) on how this allows us to do pixel perfect
rendering. At the time of writing, however this has very limited support.
The advices on feature detection for `border-box` apply here too.

### Custom Rounding

By default this hook passes the measured values through `Math.round()`, to avoid re-rendering on every subpixel changes.

If this is not what you want, then you can provide your own function:

**Rounding Down Reported Values**

```tsx
const { ref, width, height } = useResizeObserver<HTMLDivElement>({
  round: Math.floor,
});
```

**Skipping Rounding**

```tsx
import React from "react";
import useResizeObserver from "use-resize-observer";

// Outside the hook to ensure this instance does not change unnecessarily.
const noop = (n) => n;

const App = () => {
  const {
    ref,
    width = 1,
    height = 1,
  } = useResizeObserver<HTMLDivElement>({ round: noop });

  return (
    <div ref={ref}>
      Size: {width}x{height}
    </div>
  );
};
```

Note that the round option is sensitive to the function reference, so make sure you either use `useCallback`
or declare your rounding function outside of the hook's function scope, if it does not rely on any hook state.
(As shown above.)

### Getting the Raw Element from the Default `RefCallback`

Note that "ref" in the above examples is a `RefCallback`, not a `RefObject`, meaning you won't be
able to access "ref.current" if you need the element itself.

To get the raw element, either you use your own RefObject (see later in this doc),
or you can merge the returned ref with one of your own:

```tsx
import React, { useCallback, useEffect, useRef } from "react";
import useResizeObserver from "use-resize-observer";
import mergeRefs from "react-merge-refs";

const App = () => {
  const { ref, width = 1, height = 1 } = useResizeObserver<HTMLDivElement>();

  const mergedCallbackRef = mergeRefs([
    ref,
    (element: HTMLDivElement) => {
      // Do whatever you want with the `element`.
    },
  ]);

  return (
    <div ref={mergedCallbackRef}>
      Size: {width}x{height}
    </div>
  );
};
```

## Passing in Your Own `ref`

You can pass in your own ref instead of using the one provided.
This can be useful if you already have a ref you want to measure.

```ts
const ref = useRef<HTMLDivElement>(null);
const { width, height } = useResizeObserver<HTMLDivElement>({ ref });
```

You can even reuse the same hook instance to measure different elements:

[CodeSandbox Demo](https://codesandbox.io/s/use-resize-observer-reusing-refs-buftd)

## Measuring a raw element

There might be situations where you have an element already that you need to measure.
`ref` now accepts elements as well, not just refs, which means that you can do this:

```ts
const { width, height } = useResizeObserver<HTMLDivElement>({
  ref: divElement,
});
```

## Using a Single Hook to Measure Multiple Refs

The hook reacts to ref changes, as it resolves it to an element to observe.
This means that you can freely change the custom `ref` option from one ref to
another and back, and the hook will start observing whatever is set in its options.

## Opting Out of (or Delaying) ResizeObserver Instantiation

In certain cases you might want to delay creating a ResizeObserver instance.

You might provide a library, that only optionally provides observation features
based on props, which means that while you have the hook within your component,
you might not want to actually initialise it.

Another example is that you might want to entirely opt out of initialising, when
you run some tests, where the environment does not provide the `ResizeObserver`.

([See discussions](https://github.com/ZeeCoder/use-resize-observer/issues/40))

You can do one of the following depending on your needs:

- Use the default `ref` RefCallback, or provide a custom ref conditionally,
  only when needed. The hook will not create a ResizeObserver instance up until
  there's something there to actually observe.
- Patch the test environment, and make a polyfill available as the ResizeObserver.
  (This assumes you don't already use the polyfilled version, which would switch
  to the polyfill when no native implementation was available.)

## The "onResize" Callback

By the default the hook will trigger a re-render on all changes to the target
element's width and / or height.

You can opt out of this behaviour, by providing an `onResize` callback function,
which'll simply receive the width and height of the element when it changes, so
that you can decide what to do with it:

```tsx
import React from "react";
import useResizeObserver from "use-resize-observer";

const App = () => {
  // width / height will not be returned here when the onResize callback is present
  const { ref } = useResizeObserver<HTMLDivElement>({
    onResize: ({ width, height }) => {
      // do something here.
    },
  });

  return <div ref={ref} />;
};
```

This callback also makes it possible to implement your own hooks that report only
what you need, for example:

- Reporting only width or height
- Throttle / debounce
- Wrap in `requestAnimationFrame`

## Hook Composition

As this hook intends to remain low-level, it is encouraged to build on top of it via hook composition, if additional features are required.

### Throttle / Debounce

You might want to receive values less frequently than changes actually occur.

[CodeSandbox Demo](https://codesandbox.io/s/use-resize-observer-throttle-and-debounce-8uvsg)

### Breakpoints

Another popular concept are breakpoints. Here is an example for a simple hook accomplishing that.

[CodeSandbox Demo](https://codesandbox.io/s/use-resize-observer-breakpoints-3hiv8)

## Defaults (SSR)

On initial mount the ResizeObserver will take a little time to report on the
actual size.

Until the hook receives the first measurement, it returns `undefined` for width
and height by default.

You can override this behaviour, which could be useful for SSR as well.

```ts
const { ref, width = 100, height = 50 } = useResizeObserver<HTMLDivElement>();
```

Here "width" and "height" will be 100 and 50 respectively, until the
ResizeObserver kicks in and reports the actual size.

## Without Defaults

If you only want real measurements (only values from the ResizeObserver without
any default values), then you can just leave defaults off:

```ts
const { ref, width, height } = useResizeObserver<HTMLDivElement>();
```

Here "width" and "height" will be undefined until the ResizeObserver takes its
first measurement.

## Container/Element Query with CSS-in-JS

It's possible to apply styles conditionally based on the width / height of an
element using a CSS-in-JS solution, which is the basic idea behind
container/element queries:

[CodeSandbox Demo](https://codesandbox.io/s/use-resize-observer-container-query-with-css-in-js-iitxl)

## Transpilation / Polyfilling

By default the library provides transpiled ES5 modules in CJS / ESM module formats.

Polyfilling is recommended to be done in the host app, and not within imported
libraries, as that way consumers have control over the exact polyfills being used.

That said, there's a [polyfilled](https://github.com/juggle/resize-observer)
CJS module that can be used for convenience:

```ts
import useResizeObserver from "use-resize-observer/polyfilled";
```

Note that using the above will use the polyfill, [even if the native ResizeObserver is available](https://github.com/juggle/resize-observer#basic-usage).

To use the polyfill as a fallback only when the native RO is unavailable, you can polyfill yourself instead,
either in your app's entry file, or you could create a local `useResizeObserver` module, like so:

```ts
// useResizeObserver.ts
import { ResizeObserver } from "@juggle/resize-observer";
import useResizeObserver from "use-resize-observer";

if (!window.ResizeObserver) {
  window.ResizeObserver = ResizeObserver;
}

export default useResizeObserver;
```

The same technique can also be used to provide any of your preferred ResizeObserver polyfills out there.

## Related

- [@zeecoder/container-query](https://github.com/ZeeCoder/container-query)
- [@zeecoder/react-resize-observer](https://github.com/ZeeCoder/react-resize-observer)

## License

MIT
