var tape = require('tape')
var zlib = require('zlib')
var concat = require('concat-stream')
var fs = require('fs')
var gunzip = require('./')

tape('deflated input', function (t) {
  fs.createReadStream(__filename)
    .pipe(zlib.createDeflate())
    .pipe(gunzip())
    .pipe(concat(function (data) {
      t.same(data, fs.readFileSync(__filename))
      t.end()
    }))
})

tape('deflated multiple times', function (t) {
  fs.createReadStream(__filename)
    .pipe(zlib.createDeflate())
    .pipe(zlib.createDeflate())
    .pipe(gunzip())
    .pipe(concat(function (data) {
      t.same(data, fs.readFileSync(__filename))
      t.end()
    }))
})

tape('gunzipped input', function (t) {
  fs.createReadStream(__filename)
    .pipe(zlib.createGzip())
    .pipe(gunzip())
    .pipe(concat(function (data) {
      t.same(data, fs.readFileSync(__filename))
      t.end()
    }))
})

tape('gunzipped multiple times', function (t) {
  fs.createReadStream(__filename)
    .pipe(zlib.createGzip())
    .pipe(zlib.createGzip())
    .pipe(gunzip())
    .pipe(concat(function (data) {
      t.same(data, fs.readFileSync(__filename))
      t.end()
    }))
})

tape('regular input', function (t) {
  fs.createReadStream(__filename)
    .pipe(gunzip())
    .pipe(concat(function (data) {
      t.same(data, fs.readFileSync(__filename))
      t.end()
    }))
})

tape('limited recursion', function (t) {
  t.plan(1)
  fs.createReadStream(__filename)
    .pipe(zlib.createGzip())
    .pipe(zlib.createGzip())
    .pipe(gunzip(1))
    .on('finish', function () {
      t.fail('should not finish')
    })
    .on('error', function (err) {
      t.same(err.message, 'Maximum recursion reached')
    })
})
