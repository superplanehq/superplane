var assert = require('assert');
var jp = require('../');

suite('stringify', function() {

  test('simple path stringifies', function() {
    var string = jp.stringify(['$', 'a', 'b', 'c']);
    assert.equal(string, '$.a.b.c');
  });

  test('numeric literals end up as subscript numbers', function() {
    var string = jp.stringify(['$', 'store', 'book', 0, 'author']);
    assert.equal(string, '$.store.book[0].author');
  });

  test('simple path with no leading root stringifies', function() {
    var string = jp.stringify(['a', 'b', 'c']);
    assert.equal(string, '$.a.b.c');
  });

  test('simple parsed path stringifies', function() {
    var path = [
      { scope: 'child', operation: 'member', expression: { type: 'identifier', value: 'a' } },
      { scope: 'child', operation: 'member', expression: { type: 'identifier', value: 'b' } },
      { scope: 'child', operation: 'member', expression: { type: 'identifier', value: 'c' } }
    ];
    var string = jp.stringify(path);
    assert.equal(string, '$.a.b.c');
  });

  test('keys with hyphens get subscripted', function() {
    var string = jp.stringify(['$', 'member-search']);
    assert.equal(string, '$["member-search"]');
  });

  test('complicated path round trips', function() {
    var pathExpression = '$..*[0:2].member["string-xyz"]';
    var path = jp.parse(pathExpression);
    var string = jp.stringify(path);
    assert.equal(string, pathExpression);
  });

  test('complicated path with filter exp round trips', function() {
    var pathExpression = '$..*[0:2].member[?(@.val > 10)]';
    var path = jp.parse(pathExpression);
    var string = jp.stringify(path);
    assert.equal(string, pathExpression);
  });

  test('throws for no input', function() {
    assert.throws(function() { jp.stringify() }, /we need a path/);
  });

});
