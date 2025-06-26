var assert = require('assert');
var jp = require('../');

var data = require('./data/store.json');

suite('orig-google-code-issues', function() {
    
  test('comma in eval', function() {
    var pathExpression = '$..book[?(@.price && ",")]'
    var results = jp.query(data, pathExpression);
    assert.deepEqual(results, data.store.book);
  });

  test('member names with dots', function() {
    var data = { 'www.google.com': 42, 'www.wikipedia.org': 190 };
    var results = jp.query(data, "$['www.google.com']");
    assert.deepEqual(results, [ 42 ]);
  });

  test('nested objects with filter', function() {
    var data = { dataResult: { object: { objectInfo: { className: "folder", typeName: "Standard Folder", id: "uniqueId" } } } };
    var results = jp.query(data, "$..object[?(@.className=='folder')]");
    assert.deepEqual(results, [ data.dataResult.object.objectInfo ]);
  });

  test('script expressions with @ char', function() {
    var data = { "DIV": [{ "@class": "value", "val": 5 }] };
    var results = jp.query(data, "$..DIV[?(@['@class']=='value')]");
    assert.deepEqual(results, data.DIV);
  });

  test('negative slices', function() {
    var results = jp.query(data, "$..book[-1:].title");
    assert.deepEqual(results, ['The Lord of the Rings']);
  });

});

