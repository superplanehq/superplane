# Pretty print object
[![License][license-image]][license-url] ![coverage-badge-green]
 
> Convert an object or array into a formatted string

This is a re-write of [stringify-object] in Typescript, modified to inline the dependencies and make it compatible with ES5 out of the box.

Useful for when you want to get the string representation of an object in a formatted way.

It also handles circular references and lets you specify quote type.


## Install

```
$ npm install @base2/pretty-print-object
```


## Usage

```js
import { prettyPrint } from '@base2/pretty-print-object';

const obj = {
    foo: 'bar',
    'arr': [1, 2, 3],
    nested: {
        hello: "world"
    }
};

const pretty = prettyPrint(obj, {
    indent: '  ',
    singleQuotes: false
});

console.log(pretty);
/*
{
    foo: "bar",
    arr: [
        1,
        2,
        3
    ],
    nested: {
        hello: "world"
    }
}
*/
```


## API

### prettyPrint(input, [options])

Circular references will be replaced with `"[Circular]"`.

Object keys are only quoted when necessary, for example, `{'foo-bar': true}`.

#### input

Type: `Object` `Array`

#### options

Type: `Object`

##### indent

Type: `string`<br>
Default: `\t`

Preferred indentation.

##### singleQuotes

Type: `boolean`<br>
Default: `true`

Set to false to get double-quoted strings.

##### filter(obj, prop)

Type: `Function`

Expected to return a `boolean` of whether to include the property `prop` of the object `obj` in the output.

##### transform(obj, prop, originalResult)

Type: `Function`<br>
Default: `undefined`

Expected to return a `string` that transforms the string that resulted from stringifying `obj[prop]`. This can be used to detect special types of objects that need to be stringified in a particular way. The `transform` function might return an alternate string in this case, otherwise returning the `originalResult`.

Here's an example that uses the `transform` option to mask fields named "password":

```js
import { prettyPrint } from '@base2/pretty-print-object';

const obj = {
    user: 'becky',
    password: 'secret'
};

const pretty = prettyPrint(obj, {
    transform: (obj, prop, originalResult) => {
        if (prop === 'password') {
            return originalResult.replace(/\w/g, '*');
        }

        return originalResult;
    }
});

console.log(pretty);
/*
{
    user: 'becky',
    password: '******'
}
*/
```


##### inlineCharacterLimit

Type: `number`

When set, will inline values up to `inlineCharacterLimit` length for the sake of more terse output.

For example, given the example at the top of the README:

```js
import { prettyPrint } from '@base2/pretty-print-object';

const obj = {
    foo: 'bar',
    'arr': [1, 2, 3],
    nested: {
        hello: "world"
    }
};

const pretty = prettyPrint(obj, {
    indent: '  ',
    singleQuotes: false,
    inlineCharacterLimit: 12
});

console.log(pretty);
/*
{
    foo: "bar",
    arr: [1, 2, 3],
    nested: {
        hello: "world"
    }
}
*/
```

As you can see, `arr` was printed as a one-liner because its string was shorter than 12 characters.

[stringify-object]: https://www.npmjs.com/package/stringify-object
[coverage-badge-green]: https://img.shields.io/badge/Coverage-100%25-brightgreen.svg
[license-url]: https://opensource.org/licenses/BSD-2-Clause
[license-image]: https://img.shields.io/badge/License-BSD%202--Clause-orange.svg
