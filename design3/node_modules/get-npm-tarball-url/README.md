# get-npm-tarball-url

[![Build Status](https://travis-ci.org/pnpm/get-npm-tarball-url.svg?branch=master)](https://travis-ci.org/pnpm/get-npm-tarball-url)

> Create the tarball URL of a npm package

## Installation

```
pnpm add get-npm-tarball-url
```

## Usage

```js
import getNpmTarballUrl from 'get-npm-tarball-url'

const url = getNpmTarballUrl('foo', '1.0.0')

console.log(url)
// 'https://registry.npmjs.org/foo/-/foo-1.0.0.tgz'
```

## Related

- [parse-npm-tarball-url](https://github.com/pnpm/parse-npm-tarball-url) - Parse a tarball URL hosted in the npm registry

## License

[MIT](LICENSE) © [Zoltan Kochan](https://www.kochan.io)
