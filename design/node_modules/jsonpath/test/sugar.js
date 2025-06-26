var assert = require('assert');
var jp = require('../');
var util = require('util');

suite('sugar', function() {

  test('parent gets us parent value', function() {
    var data = { a: 1, b: 2, c: 3, z: { a: 100, b: 200 } };
    var parent = jp.parent(data, '$.z.b');
    assert.equal(parent, data.z);
  });

  test('apply method sets values', function() {
    var data = { a: 1, b: 2, c: 3, z: { a: 100, b: 200 } };
    jp.apply(data, '$..a', function(v) { return v + 1 });
    assert.equal(data.a, 2);
    assert.equal(data.z.a, 101);
  });

  test('apply method applies survives structural changes', function() {
    var data = {a: {b: [1, {c: [2,3]}]}};
    jp.apply(data, '$..*[?(@.length > 1)]', function(array) {
      return array.reverse();
    });
    assert.deepEqual(data.a.b, [{c: [3, 2]}, 1]);
  });

  test('value method gets us a value', function() {
    var data = { a: 1, b: 2, c: 3, z: { a: 100, b: 200 } };
    var b = jp.value(data, '$..b')
    assert.equal(b, data.b);
  });

  test('value method sets us a value', function() {
    var data = { a: 1, b: 2, c: 3, z: { a: 100, b: 200 } };
    var b = jp.value(data, '$..b', '5000')
    assert.equal(b, 5000);
    assert.equal(data.b, 5000);
  });

  test('value method sets new key and value', function() {
    var data = {};
    var a = jp.value(data, '$.a', 1);
    var c = jp.value(data, '$.b.c', 2);
    assert.equal(a, 1);
    assert.equal(data.a, 1);
    assert.equal(c, 2);
    assert.equal(data.b.c, 2);
  });

  test('value method sets new array value', function() {
    var data = {};
    var v1 = jp.value(data, '$.a.d[0]', 4);
    var v2 = jp.value(data, '$.a.d[1]', 5);
    assert.equal(v1, 4);
    assert.equal(v2, 5);
    assert.deepEqual(data.a.d, [4, 5]);
  });

  test('value method sets non-literal key', function() {
    var data = { "list": [ { "index": 0, "value": "default" }, { "index": 1, "value": "default" } ] };
    jp.value(data, '$.list[?(@.index == 1)].value', "test");
    assert.equal(data.list[1].value, "test");
  });

  test('paths with a count gets us back count many paths', function() {
    data = [ { a: [ 1, 2, 3 ], b: [ -1, -2, -3 ] }, { } ]
    paths = jp.paths(data, '$..*', 3)
    assert.deepEqual(paths, [ ['$', '0'], ['$', '1'], ['$', '0', 'a'] ]);
  });

});
