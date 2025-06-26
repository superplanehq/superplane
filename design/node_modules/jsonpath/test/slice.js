var assert = require('assert');
var slice = require('../lib/slice');

var data = ['a', 'b', 'c', 'd', 'e', 'f'];

suite('slice', function() {

  test('no params yields copy', function() {
    assert.deepEqual(slice(data), data);
  });

  test('no end param defaults to end', function() {
    assert.deepEqual(slice(data, 2), data.slice(2));
  });

  test('zero end param yields empty', function() {
    assert.deepEqual(slice(data, 0, 0), []);
  });

  test('first element with explicit params', function() {
    assert.deepEqual(slice(data, 0, 1, 1), ['a']);
  });

  test('last element with explicit params', function() {
    assert.deepEqual(slice(data, -1, 6), ['f']);
  });

  test('empty extents and negative step reverses', function() {
    assert.deepEqual(slice(data, null, null, -1), ['f', 'e', 'd', 'c', 'b', 'a']);
  });

  test('negative step partial slice', function() {
    assert.deepEqual(slice(data, 4, 2, -1), ['e', 'd']);
  });

  test('negative step partial slice no start defaults to end', function() {
    assert.deepEqual(slice(data, null, 2, -1), ['f', 'e', 'd']);
  });

  test('extents clamped end', function() {
    assert.deepEqual(slice(data, null, 100), data);
  });

  test('extents clamped beginning', function() {
    assert.deepEqual(slice(data, -100, 100), data);
  });

  test('backwards extents yields empty', function() {
    assert.deepEqual(slice(data, 2, 1), []);
  });

  test('zero step gets shot down', function() {
    assert.throws(function() { slice(data, null, null, 0) });
  });

});

