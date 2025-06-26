var assert = require('assert');
var jp = require('../');
var util = require('util');

suite('parse', function() {

  test('should parse root-only', function() {
    var path = jp.parse('$');
    assert.deepEqual(path, [ { expression: { type: 'root', value: '$' } } ]);
  });

  test('parse path for store', function() {
    var path = jp.parse('$.store');
    assert.deepEqual(path, [
      { expression: { type: 'root', value: '$' } },
      { operation: 'member', scope: 'child', expression: { type: 'identifier', value: 'store' } }
    ])
  });

  test('parse path for the authors of all books in the store', function() {
    var path = jp.parse('$.store.book[*].author');
    assert.deepEqual(path, [
      { expression: { type: 'root', value: '$' } },
      { operation: 'member', scope: 'child', expression: { type: 'identifier', value: 'store' } },
      { operation: 'member', scope: 'child', expression: { type: 'identifier', value: 'book' } },
      { operation: 'subscript', scope: 'child', expression: { type: 'wildcard', value: '*' } },
      { operation: 'member', scope: 'child', expression: { type: 'identifier', value: 'author' } }
    ])
  });

  test('parse path for all authors', function() {
    var path = jp.parse('$..author');
    assert.deepEqual(path, [
      { expression: { type: 'root', value: '$' } },
      { operation: 'member', scope: 'descendant', expression: { type: 'identifier', value: 'author' } }
    ])
  });

  test('parse path for all authors via subscript descendant string literal', function() {
    var path = jp.parse("$..['author']");
    assert.deepEqual(path, [
      { expression: { type: 'root', value: '$' } },
      { operation: 'subscript', scope: 'descendant', expression: { type: 'string_literal', value: 'author' } }
    ])
  });

  test('parse path for all things in store', function() {
    var path = jp.parse('$.store.*');
    assert.deepEqual(path, [
      { expression: { type: 'root', value: '$' } },
      { operation: 'member', scope: 'child', expression: { type: 'identifier', value: 'store' } },
      { operation: 'member', scope: 'child', expression: { type: 'wildcard', value: '*' } }
    ])
  });

  test('parse path for price of everything in the store', function() {
    var path = jp.parse('$.store..price');
    assert.deepEqual(path, [
      { expression: { type: 'root', value: '$' } },
      { operation: 'member', scope: 'child', expression: { type: 'identifier', value: 'store' } },
      { operation: 'member', scope: 'descendant', expression: { type: 'identifier', value: 'price' } }
    ])
  });

  test('parse path for the last book in order via expression', function() {
    var path = jp.parse('$..book[(@.length-1)]');
    assert.deepEqual(path, [
      { expression: { type: 'root', value: '$' } },
      { operation: 'member', scope: 'descendant', expression: { type: 'identifier', value: 'book' } },
      { operation: 'subscript', scope: 'child', expression: { type: 'script_expression', value: '(@.length-1)' } }
    ])
  });

  test('parse path for the first two books via union', function() {
    var path = jp.parse('$..book[0,1]');

    assert.deepEqual(path, [
      { expression: { type: 'root', value: '$' } },
      { operation: 'member', scope: 'descendant', expression: { type: 'identifier', value: 'book' } },
      { operation: 'subscript', scope: 'child', expression: { type: 'union', value: [ { expression: { type: 'numeric_literal', value: '0' } }, { expression: { type: 'numeric_literal', value: '1' } } ] } }
    ])
  });

  test('parse path for the first two books via slice', function() {
    var path = jp.parse('$..book[0:2]');
    assert.deepEqual(path, [
      { expression: { type: 'root', value: '$' } },
      { operation: 'member', scope: 'descendant', expression: { type: 'identifier', value: 'book' } },
      { operation: 'subscript', scope: 'child', expression: { type: 'slice', value: '0:2' } }
    ])
  });

  test('parse path to filter all books with isbn number', function() {
    var path = jp.parse('$..book[?(@.isbn)]');
    assert.deepEqual(path, [
      { expression: { type: 'root', value: '$' } },
      { operation: 'member', scope: 'descendant', expression: { type: 'identifier', value: 'book' } },
      { operation: 'subscript', scope: 'child', expression: { type: 'filter_expression', value: '?(@.isbn)' } }
    ])
  });

  test('parse path to filter all books with a price less than 10', function() {
    var path = jp.parse('$..book[?(@.price<10)]');
    assert.deepEqual(path, [
      { expression: { type: 'root', value: '$' } },
      { operation: 'member', scope: 'descendant', expression: { type: 'identifier', value: 'book' } },
      { operation: 'subscript', scope: 'child', expression: { type: 'filter_expression', value: '?(@.price<10)' } }
    ])
  });

  test('parse path to match all elements', function() {
    var path = jp.parse('$..*');
    assert.deepEqual(path, [
      { expression: { type: 'root', value: '$' } },
      { operation: 'member', scope: 'descendant', expression: { type: 'wildcard', value: '*' } }
    ])
  });

  test('parse path with leading member', function() {
    var path = jp.parse('store');
    assert.deepEqual(path, [
      { operation: 'member', scope: 'child', expression: { type: 'identifier', value: 'store' } }
    ])
  });

  test('parse path with leading member and followers', function() {
    var path = jp.parse('Request.prototype.end');
    assert.deepEqual(path, [
      { operation: 'member', scope: 'child', expression: { type: 'identifier', value: 'Request' } },
      { operation: 'member', scope: 'child', expression: { type: 'identifier', value: 'prototype' } },
      { operation: 'member', scope: 'child', expression: { type: 'identifier', value: 'end' } }
    ])
  });

  test('parser ast is reinitialized after parse() throws', function() {
    assert.throws(function() { var path = jp.parse('store.book...') })
    var path = jp.parse('$..price');
    assert.deepEqual(path, [
      { "expression": { "type": "root", "value": "$" } },
      { "expression": { "type": "identifier", "value": "price" }, "operation": "member", "scope": "descendant"}
    ])
  });

});

suite('parse-negative', function() {

  test('parse path with leading member component throws', function() {
    assert.throws(function(e) { var path = jp.parse('.store') }, /Expecting 'DOLLAR'/)
  });

  test('parse path with leading descendant member throws', function() {
    assert.throws(function() { var path = jp.parse('..store') }, /Expecting 'DOLLAR'/)
  });

  test('leading script throws', function() {
    assert.throws(function() { var path = jp.parse('()') }, /Unrecognized text/)
  });

  test('first time friendly error', function() {
    assert.throws(function() { (new jp.JSONPath).parse('$...') }, /Expecting 'STAR'/)
  });

});
