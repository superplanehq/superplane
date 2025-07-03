# download-tarball

Download a tarball (optionally gzipped) to a folder &amp; extract it in the process. Uses the wonderful &amp; super quick tar-fs &amp; gunzip-maybe srcraries.

## Installation

This is a [Node.js](https://nodejs.org/) module available through the 
[npm registry](https://www.npmjs.com/). It can be installed using the 
[`npm`](https://docs.npmjs.com/getting-started/installing-npm-packages-locally)
or 
[`yarn`](https://yarnpkg.com/en/)
command line tools.

```sh
npm install download-tarball --save
```

## Usage

```js
/* eslint-disable import/no-extraneous-dependencies */

import download from 'download-tarball';

download({
  url: 'http://link-to-tarball/file.tar.gz',
  dir: '/dir/where/file/will/be/downloaded'
}).then(() => {
  console.log('file is now downloaded!');
}).catch(err => {
  console.log('oh crap the file could not be downloaded properly');
  console.log(err);
});

download({
  url: 'http://link-to-tarball/file.tar.gz',
  dir: '/dir/where/file/will/be/downloaded',
  // custom options that will be forwarded to got.stream(..., opts) can also be set
  gotOpts: {
    headers: {
      beep: 'boop'
    }
  }
}).then(() => {
  console.log('file is now downloaded!');
}).catch(err => {
  console.log('oh crap the file could not be downloaded properly');
  console.log(err);
});

download({
  // neat, tar files works as well!
  url: 'http://link-to-tarball/file.tar',
  dir: '/dir/where/file/will/be/downloaded'
}).then(() => {
  console.log('file is now downloaded!');
}).catch(err => {
  console.log('oh crap the file could not be downloaded properly');
  console.log(err);
});

```

## Tests

```sh
npm install
npm test
```

## Dependencies

- [gunzip-maybe](https://github.com/mafintosh/gunzip-maybe): Transform stream that gunzips its input if it is gzipped and just echoes it if not
- [pump](https://github.com/mafintosh/pump): pipe streams together and close all of them if one of them closes
- [tar-fs](https://github.com/mafintosh/tar-fs): filesystem bindings for tar-stream

## License

MIT
