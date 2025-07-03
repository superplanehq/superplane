# tempy

> Get a random temporary file or directory path

## Install

```
$ npm install tempy
```

## Usage

```js
const tempy = require('tempy');

tempy.file();
//=> '/private/var/folders/3x/jf5977fn79jbglr7rk0tq4d00000gn/T/4f504b9edb5ba0e89451617bf9f971dd'

tempy.file({extension: 'png'});
//=> '/private/var/folders/3x/jf5977fn79jbglr7rk0tq4d00000gn/T/a9fb0decd08179eb6cf4691568aa2018.png'

tempy.file({name: 'unicorn.png'});
//=> '/private/var/folders/3x/jf5977fn79jbglr7rk0tq4d00000gn/T/f7f62bfd4e2a05f1589947647ed3f9ec/unicorn.png'

tempy.directory();
//=> '/private/var/folders/3x/jf5977fn79jbglr7rk0tq4d00000gn/T/2f3d094aec2cb1b93bb0f4cffce5ebd6'

tempy.directory({prefix: 'name'});
//=> '/private/var/folders/3x/jf5977fn79jbglr7rk0tq4d00000gn/T/name_3c085674ad31223b9653c88f725d6b41'
```

## API

### tempy.file(options?)

Get a temporary file path you can write to.

### tempy.file.task(callback, options?)

The `callback` resolves with a temporary file path you can write to. The file is automatically cleaned up after the callback is executed. Returns a promise that resolves with the return value of the callback after it is executed and the file is cleaned up.

#### callback

Type: `(tempPath: string) => void`

A callback that is executed with the temp file path. Can be asynchronous.

#### options

Type: `object`

*You usually won't need either the `extension` or `name` option. Specify them only when actually needed.*

##### extension

Type: `string`

File extension.

##### name

Type: `string`

Filename. Mutually exclusive with the `extension` option.

### tempy.directory(options?)

Get a temporary directory path. The directory is created for you.

### tempy.directory.task(callback, options?)

The `callback` resolves with a temporary directory path you can write to. The directory is automatically cleaned up after the callback is executed. Returns a promise that resolves with the return value of the callback after it is executed and the directory is cleaned up.

##### callback

Type: `(tempPath: string) => void`

A callback that is executed with the temp directory path. Can be asynchronous.

#### options

Type: `Object`

##### prefix

Type: `string`

Directory prefix.

Useful for testing by making it easier to identify cache directories that are created.

*You usually won't need this option. Specify it only when actually needed.*

### tempy.write(fileContent, options?)

Write data to a random temp file.

### tempy.write.task(fileContent, callback, options?)

Write data to a random temp file. The file is automatically cleaned up after the callback is executed. Returns a promise that resolves with the return value of the callback after it is executed and the file is cleaned up.

##### fileContent

Type: `string | Buffer | TypedArray | DataView | stream.Readable`

Data to write to the temp file.

##### callback

Type: `(tempPath: string) => void`

A callback that is executed with the temp file path. Can be asynchronous.

##### options

See [options](#options).

### tempy.writeSync(fileContent, options?)

Synchronously write data to a random temp file.

##### fileContent

Type: `string | Buffer | TypedArray | DataView`

Data to write to the temp file.

##### options

See [options](#options).

### tempy.root

Get the root temporary directory path. For example: `/private/var/folders/3x/jf5977fn79jbglr7rk0tq4d00000gn/T`
