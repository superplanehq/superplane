# Promise version of glob

Match files using the patterns the shell uses, like stars and stuff.

[![license][license-img]][license-url]
[![release][release-img]][release-url]
[![super linter][super-linter-img]][super-linter-url]
[![test][test-img]][test-url]
[![semantic][semantic-img]][semantic-url]

> ***Note**: This is just a [`Promise`][] wrapped version of [`glob`][]*

## Install

``` bash
$ npm install glob-promise glob
```

###### NOTE:

[`glob`][] is set as a `peerDependency` in [`package.json`][]

-   `npm` &lt;= `2` will automatically install `peerDependencies` if they are not explicitly depended upon higher in the dependency tree.
-   `npm` &gt;= 3 will no longer automatically install `peerDependencies`.

You will need to manually add `glob` as a dependency to your project for `glob-promise` to work.

## API

### `glob(pattern [, options])`

Alias for `glob.promise`

### `glob.promise(pattern [, options])`

*pattern*: `String` (glob pattern)
*options*: `Object` or `String`
Return: `Object` ([Promise][`Promise`])

When it finishes, it will be [*fulfilled*][] with an `Array` of filenames as its first argument.

When it fails to read the files, it will be [*rejected*][] with an error as its first argument.

``` js
glob('**/*')
  .then(function(contents) {
    contents; //=> ['lorem', 'ipsum', 'dolor']
  });

glob('{foo,bar.baz}.txt', { nobrace: true })
  .then(function(contents) {
    contents; //=> []
  });
```

### `glob.glob(pattern [, options], cb)`

> see [`glob`][1]

### `glob.sync(pattern [, options])`

> see [`glob.sync()`][]

### `glob.hasMagic(pattern, [options])`

> see [`glob.hasMagic()`][]

### `Class: glob.Glob`

> see [`Glob`][2]

#### options

The option object will be directly passed to [glob][].

  [`Promise`]: http://promisesaplus.com/
  [`glob`]: https://github.com/isaacs/node-glob
  [`package.json`]: package.json
  [*fulfilled*]: http://promisesaplus.com/#point-26
  [*rejected*]: http://promisesaplus.com/#point-30
  [1]: https://github.com/isaacs/node-glob#globpattern-options-cb
  [`glob.sync()`]: https://github.com/isaacs/node-glob#globsyncpattern-options
  [`glob.hasMagic()`]: https://github.com/isaacs/node-glob#globhasmagicpattern-options
  [2]: https://github.com/isaacs/node-glob#class-globglob
  [glob]: https://github.com/isaacs/node-glob#options

----
> Author: [Ahmad Nassri](https://www.ahmadnassri.com/) &bull;
> Twitter: [@AhmadNassri](https://twitter.com/AhmadNassri)

[license-url]: LICENSE
[license-img]: https://badgen.net/github/license/ahmadnassri/node-glob-promise

[release-url]: https://github.com/ahmadnassri/node-glob-promise/releases
[release-img]: https://badgen.net/github/release/ahmadnassri/node-glob-promise

[super-linter-url]: https://github.com/ahmadnassri/node-glob-promise/actions?query=workflow%3Asuper-linter
[super-linter-img]: https://github.com/ahmadnassri/node-glob-promise/workflows/super-linter/badge.svg

[test-url]: https://github.com/ahmadnassri/node-glob-promise/actions?query=workflow%3Atest
[test-img]: https://github.com/ahmadnassri/node-glob-promise/workflows/test/badge.svg

[semantic-url]: https://github.com/ahmadnassri/node-glob-promise/actions?query=workflow%3Arelease
[semantic-img]: https://badgen.net/badge/📦/semantically%20released/blue
