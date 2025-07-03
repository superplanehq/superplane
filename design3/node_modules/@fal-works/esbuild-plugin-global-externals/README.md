# esbuild-plugin-global-externals

[esbuild](https://esbuild.github.io/) plugin for replacing imports with global variables.

Similar to (but not the same as) `output.globals` option of [Rollup](https://rollupjs.org/). See also: [evanw/esbuild#337](https://github.com/evanw/esbuild/issues/337)

Originally developed with esbuild v0.8.40, and should work with higher versions as well.


## TL;DR

This:

```js
import p5 from "p5";
new p5();
```

transpiles to:

```js
// Here a variable `p5` should be defined globally (typically via UMD)
var p5_default = p5;
new p5_default();
```


## Usage example

```js
import { globalExternals } from "@fal-works/esbuild-plugin-global-externals";

/** Mapping from module paths to global variables */
const globals = {
  jquery: "$"
};

esbuild.build({
  entryPoints: ["src/main.js"],
  outfile: "dist/bundle.js",
  bundle: true,
  plugins: [globalExternals(globals)],
});
```

If you prefer `RegExp` use `globalExternalsWithRegExp()` instead, however note that the `RegExp` you'll pass should be valid in Go language as well.


## Configuration

Instead of simply providing variable names only, you can pass `ModuleInfo` objects to configure the loading behavior for each module.

### Module type

Either `"esm"` (default) or `"cjs"`.

This determines the internal behavior of this plugin when loading modules (which affects the code after bundling as well).

```js
globalExternals({
  "someModulePath": {
    varName: "someGlobalVar",
    type: "cjs"
  }
})
```

### Named/default export

If `type: "esm"` and also if you're doing named import with some modules in question, you have to tell which variables from each module should be importable.

Additionally, if you don't do default import/export, you can prevent emitting code for this with `defaultExport: false`.

Both have no effects if `type: "cjs"` (which should work fine with named imports as well).

```js
globalExternals({
  "someModulePath": {
    varName: "someGlobalVar",
    namedExports: ["someExportedVar"],
    defaultExport: false
  }
})
```

The example above enables the following even if `type: "esm"`:

```js
import { someExportedVar } from "someModulePath";
```

which transpiles to:

```js
var someExportedVar = someGlobalVar.someExportedVar;
```
