# esbuild-plugin-alias

[![npm](https://img.shields.io/npm/v/esbuild-plugin-alias.svg)](https://www.npmjs.com/package/esbuild-plugin-alias)

[esbuild](https://github.com/evanw/esbuild) plugin for path aliases.


## Rationale

Sometimes it's useful to have dynamic imports that resolves into different files depending on some conditions 
(e.g. env variables).


## Installation

```sh
npm install --save-dev esbuild-plugin-alias
```


## Usage

Define plugin in the `plugins` section of esbuild config like this:

```js
const esbuild = require('esbuild');
const alias = require('esbuild-plugin-alias');

esbuild.build({
  // ...
  plugins: [
    alias({
      'imported-path': '/home/user/lib/src/resolved-path',
    }),
  ],
})
```

**Note:** esbuild requires resolved paths to be absolute. So, make sure that values in plugin's config object are
absolute paths.

If you need to find a path to an installed dep, you may use `require.resolve`. E.g.:

```js
alias({
  'react-dom': process.env.NODE_ENV === 'dev' 
    ? require.resolve('@hot-loader/react-dom')
    : require.resolve('react-dom'),
}),
```


## Example

Having this input file:

```js
// src/app.js
import settings from 'settings.env';

console.log(settings);
```

And esbuild config like this:

```js
// config/build.js
const path = require('path');
const esbuild = require('esbuild');
const alias = require('esbuild-plugin-alias');

esbuild.build({
  entryPoints: ['in.js'],
  bundle: true,
  outfile: 'out.js',
  plugins: [
    alias({
      'settings.env': path.resolve(__dirname, `../src/settings.${process.env.NODE_ENV}.js`),
    }),
  ],
}).catch(err => process.exit(1));
```

You will get `src/settings.dev.js` loaded instead of `settings.env` when `NODE_ENV` equals `dev`.

Check [test/](test) for more detailed example.
