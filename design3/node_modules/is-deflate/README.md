# is-deflate

Check if a given `Buffer` or `Uint8Array` is
[deflate](https://en.wikipedia.org/wiki/DEFLATE) compressed.

[![Build status](https://travis-ci.org/watson/is-deflate.svg?branch=master)](https://travis-ci.org/watson/is-deflate)
[![js-standard-style](https://img.shields.io/badge/code%20style-standard-brightgreen.svg?style=flat)](https://github.com/feross/standard)

## Installation

```
npm install is-deflate --save
```

## Usage

```js
var fs = require('fs')
var zlib = require('zlib')
var isDeflate = require('is-deflate')

var buf = fs.readFileSync('my-file')

if (isDeflate(buf)) {
  zlib.inflate(buf, function (err, data) {
    if (err) throw err
    console.log(data)
  })
}
```

## License

MIT
