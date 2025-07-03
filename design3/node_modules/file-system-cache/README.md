[![ci(esm)](https://github.com/philcockfield/file-system-cache/actions/workflows/node.esm.yml/badge.svg)](https://github.com/philcockfield/file-system-cache/actions/workflows/node.esm.yml)

# file-system-cache

A super-fast, promise-based cache that reads and writes to the file-system.


## Installation

    npm install --save file-system-cache

## Usage (API)

Create an instance of the cache optionally giving it a folder location to store files within.

```js
import Cache from "file-system-cache";

const cache = Cache({
  basePath: "./.cache", // Optional. Path where cache files are stored (default).
  ns: "my-namespace",    // Optional. A grouping namespace for items.
  ttl: 60               // Optional. A time-to-live that determines how long the cache item is valid, in seconds.
});
```

> Reference `default` for CommonJS, e.g: `const Cache = require('file-system-cache').default
`

The optional `ns` ("namespace") allows for multiple groupings of files to reside within the one cache folder.  When you have multiple caches with different namespaces you can re-use the same keys and they will not collide.

The optional `ttl` ("time to live") allows you to set a default expiration for the cache key, in seconds. For example if you
set this value to 60 then cache hits to any cache key made beyond the limit of that 60 seconds will result in cache misses.

### get(key, defaultValue)
Retrieves the contents of a cached file.

```js
cache.get("foo")
  .then(result => /* Success */)
  .catch(err => /* Fail */);
```

or use modern `async/await` syntactic sugar of course:

```js
const value = await cache.get("foo");
```

Use `getSync` for a synchronous version.  
Pass a `defaultValue` parameter to use if the key does not exist within the cache.


### set(key, value, ttl)
Write a value to the cache.

```js
/* This will apply the default TTL to this cache key */
cache.set("foo", "...value...")
  .then(result => /* Success */)

/* This will set a specific TTL for this cache key */
cache.set("bar", "...baz", 60)
  .then(result => /* Success */)
```

Value types are stored and respected on subsequent calls to `get`.  For examples, passing in Object will return that in its parsed object state.

Use `setSync` for a synchronous version.

### remove(key)
Deletes the specified cache item from the file-system.
```js
cache.remove("foo")
  .then(() => /* Removed */)
```

### clear()
Clears all cached items from the file-system.
```js
cache.clear()
  .then(() => /* All items deleted */)
```


### save()
Saves (sets) several items to the cache in one operation.
```js
cache.save([{ key:"one", value:"hello" }, { key:"two", value:222 }])
  .then(result => /* All items saved. */)
```

### load()
Loads all files within the cache's namespace.
```js
cache.load()
  .then(result => /* The complete of cached files (for the ns). */)
```



## Test
    # Run tests.
    npm test

