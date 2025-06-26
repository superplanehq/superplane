var assert = require('assert');
var jp = require('../');

var data = require('./data/store.json');

suite('query', function() {

  test('first-level member', function() {
    var results = jp.nodes(data, '$.store');
    assert.deepEqual(results, [ { path: ['$', 'store'], value: data.store } ]);
  });

  test('authors of all books in the store', function() {
    var results = jp.nodes(data, '$.store.book[*].author');
    assert.deepEqual(results, [
      { path: ['$', 'store', 'book', 0, 'author'], value: 'Nigel Rees' },
      { path: ['$', 'store', 'book', 1, 'author'], value: 'Evelyn Waugh' },
      { path: ['$', 'store', 'book', 2, 'author'], value: 'Herman Melville' },
      { path: ['$', 'store', 'book', 3, 'author'], value: 'J. R. R. Tolkien' }
    ]);
  });

  test('all authors', function() {
    var results = jp.nodes(data, '$..author');
    assert.deepEqual(results, [
      { path: ['$', 'store', 'book', 0, 'author'], value: 'Nigel Rees' },
      { path: ['$', 'store', 'book', 1, 'author'], value: 'Evelyn Waugh' },
      { path: ['$', 'store', 'book', 2, 'author'], value: 'Herman Melville' },
      { path: ['$', 'store', 'book', 3, 'author'], value: 'J. R. R. Tolkien' }
    ]);
  });

  test('all authors via subscript descendant string literal', function() {
    var results = jp.nodes(data, "$..['author']");
    assert.deepEqual(results, [
      { path: ['$', 'store', 'book', 0, 'author'], value: 'Nigel Rees' },
      { path: ['$', 'store', 'book', 1, 'author'], value: 'Evelyn Waugh' },
      { path: ['$', 'store', 'book', 2, 'author'], value: 'Herman Melville' },
      { path: ['$', 'store', 'book', 3, 'author'], value: 'J. R. R. Tolkien' }
    ]);
  });

  test('all things in store', function() {
    var results = jp.nodes(data, '$.store.*');
    assert.deepEqual(results, [
      { path: ['$', 'store', 'book'], value: data.store.book },
      { path: ['$', 'store', 'bicycle'], value: data.store.bicycle }
    ]);
  });

  test('price of everything in the store', function() {
    var results = jp.nodes(data, '$.store..price');
    assert.deepEqual(results, [
      { path: ['$', 'store', 'book', 0, 'price'], value: 8.95 },
      { path: ['$', 'store', 'book', 1, 'price'], value: 12.99 },
      { path: ['$', 'store', 'book', 2, 'price'], value: 8.99 },
      { path: ['$', 'store', 'book', 3, 'price'], value: 22.99 },
      { path: ['$', 'store', 'bicycle', 'price'], value: 19.95 }
    ]);
  });

  test('last book in order via expression', function() {
    var results = jp.nodes(data, '$..book[(@.length-1)]');
    assert.deepEqual(results, [ { path: ['$', 'store', 'book', 3], value: data.store.book[3] }]);
  });

  test('first two books via union', function() {
    var results = jp.nodes(data, '$..book[0,1]');
    assert.deepEqual(results, [
      { path: ['$', 'store', 'book', 0], value: data.store.book[0] },
      { path: ['$', 'store', 'book', 1], value: data.store.book[1] }
    ]);
  });

  test('first two books via slice', function() {
    var results = jp.nodes(data, '$..book[0:2]');
    assert.deepEqual(results, [
      { path: ['$', 'store', 'book', 0], value: data.store.book[0] },
      { path: ['$', 'store', 'book', 1], value: data.store.book[1] }
    ]);
  });

  test('filter all books with isbn number', function() {
    var results = jp.nodes(data, '$..book[?(@.isbn)]');
    assert.deepEqual(results, [
      { path: ['$', 'store', 'book', 2], value: data.store.book[2] },
      { path: ['$', 'store', 'book', 3], value: data.store.book[3] }
    ]);
  });

  test('filter all books with a price less than 10', function() {
    var results = jp.nodes(data, '$..book[?(@.price<10)]');
    assert.deepEqual(results, [
      { path: ['$', 'store', 'book', 0], value: data.store.book[0] },
      { path: ['$', 'store', 'book', 2], value: data.store.book[2] }
    ]);
  });

  test('first ten of all elements', function() {
    var results = jp.nodes(data, '$..*', 10);
    assert.deepEqual(results, [
      { path: [ '$', 'store' ], value: data.store },
      { path: [ '$', 'store', 'book' ], value: data.store.book },
      { path: [ '$', 'store', 'bicycle' ], value: data.store.bicycle },
      { path: [ '$', 'store', 'book', 0 ], value: data.store.book[0] },
      { path: [ '$', 'store', 'book', 1 ], value: data.store.book[1] },
      { path: [ '$', 'store', 'book', 2 ], value: data.store.book[2] },
      { path: [ '$', 'store', 'book', 3 ], value: data.store.book[3] },
      { path: [ '$', 'store', 'book', 0, 'category' ], value: 'reference' },
      { path: [ '$', 'store', 'book', 0, 'author' ], value: 'Nigel Rees' },
      { path: [ '$', 'store', 'book', 0, 'title' ], value: 'Sayings of the Century' }
    ])
  });

  test('all elements', function() {
    var results = jp.nodes(data, '$..*');

    assert.deepEqual(results, [
      { path: [ '$', 'store' ], value: data.store },
      { path: [ '$', 'store', 'book' ], value: data.store.book },
      { path: [ '$', 'store', 'bicycle' ], value: data.store.bicycle },
      { path: [ '$', 'store', 'book', 0 ], value: data.store.book[0] },
      { path: [ '$', 'store', 'book', 1 ], value: data.store.book[1] },
      { path: [ '$', 'store', 'book', 2 ], value: data.store.book[2] },
      { path: [ '$', 'store', 'book', 3 ], value: data.store.book[3] },
      { path: [ '$', 'store', 'book', 0, 'category' ], value: 'reference' },
      { path: [ '$', 'store', 'book', 0, 'author' ], value: 'Nigel Rees' },
      { path: [ '$', 'store', 'book', 0, 'title' ], value: 'Sayings of the Century' },
      { path: [ '$', 'store', 'book', 0, 'price' ], value: 8.95 },
      { path: [ '$', 'store', 'book', 1, 'category' ], value: 'fiction' },
      { path: [ '$', 'store', 'book', 1, 'author' ], value: 'Evelyn Waugh' },
      { path: [ '$', 'store', 'book', 1, 'title' ], value: 'Sword of Honour' },
      { path: [ '$', 'store', 'book', 1, 'price' ], value: 12.99 },
      { path: [ '$', 'store', 'book', 2, 'category' ], value: 'fiction' },
      { path: [ '$', 'store', 'book', 2, 'author' ], value: 'Herman Melville' },
      { path: [ '$', 'store', 'book', 2, 'title' ], value: 'Moby Dick' },
      { path: [ '$', 'store', 'book', 2, 'isbn' ], value: '0-553-21311-3' },
      { path: [ '$', 'store', 'book', 2, 'price' ], value: 8.99 },
      { path: [ '$', 'store', 'book', 3, 'category' ], value: 'fiction' },
      { path: [ '$', 'store', 'book', 3, 'author' ], value: 'J. R. R. Tolkien' },
      { path: [ '$', 'store', 'book', 3, 'title' ], value: 'The Lord of the Rings' },
      { path: [ '$', 'store', 'book', 3, 'isbn' ], value: '0-395-19395-8' },
      { path: [ '$', 'store', 'book', 3, 'price' ], value: 22.99 },
      { path: [ '$', 'store', 'bicycle', 'color' ], value: 'red' },
      { path: [ '$', 'store', 'bicycle', 'price' ], value: 19.95 }
    ]);
  });

  test('all elements via subscript wildcard', function() {
    var results = jp.nodes(data, '$..*');
    assert.deepEqual(jp.nodes(data, '$..[*]'), jp.nodes(data, '$..*'));
  });

  test('object subscript wildcard', function() {
    var results = jp.query(data, '$.store[*]');
    assert.deepEqual(results, [ data.store.book, data.store.bicycle ]);
  });

  test('no match returns empty array', function() {
    var results = jp.nodes(data, '$..bookz');
    assert.deepEqual(results, []);
  });

  test('member numeric literal gets first element', function() {
    var results = jp.nodes(data, '$.store.book.0');
    assert.deepEqual(results, [ { path: [ '$', 'store', 'book', 0 ], value: data.store.book[0] } ]);
  });

  test('member numeric literal matches string-numeric key', function() {
    var data = { authors: { '1': 'Herman Melville', '2': 'J. R. R. Tolkien' } };
    var results = jp.nodes(data, '$.authors.1');
    assert.deepEqual(results, [ { path: [ '$', 'authors', 1 ], value: 'Herman Melville' } ]);
  });

  test('descendant numeric literal gets first element', function() {
    var results = jp.nodes(data, '$.store.book..0');
    assert.deepEqual(results, [ { path: [ '$', 'store', 'book', 0 ], value: data.store.book[0] } ]);
  });

  test('root element gets us original obj', function() {
    var results = jp.nodes(data, '$');
    assert.deepEqual(results, [ { path: ['$'], value: data } ]);
  });

  test('subscript double-quoted string', function() {
    var results = jp.nodes(data, '$["store"]');
    assert.deepEqual(results, [ { path: ['$', 'store'], value: data.store} ]);
  });

  test('subscript single-quoted string', function() {
    var results = jp.nodes(data, "$['store']");
    assert.deepEqual(results, [ { path: ['$', 'store'], value: data.store} ]);
  });

  test('leading member component', function() {
    var results = jp.nodes(data, "store");
    assert.deepEqual(results, [ { path: ['$', 'store'], value: data.store} ]);
  });

  test('union of three array slices', function() {
    var results = jp.query(data, "$.store.book[0:1,1:2,2:3]");
    assert.deepEqual(results, data.store.book.slice(0,3));
  });

  test('slice with step > 1', function() {
    var results = jp.query(data, "$.store.book[0:4:2]");
    assert.deepEqual(results, [ data.store.book[0], data.store.book[2]]);
  });

  test('union of subscript string literal keys', function() {
    var results = jp.nodes(data, "$.store['book','bicycle']");
    assert.deepEqual(results, [
      { path: ['$', 'store', 'book'], value: data.store.book },
      { path: ['$', 'store', 'bicycle'], value: data.store.bicycle },
    ]);
  });

  test('union of subscript string literal three keys', function() {
    var results = jp.nodes(data, "$.store.book[0]['title','author','price']");
    assert.deepEqual(results, [
      { path: ['$', 'store', 'book', 0, 'title'], value: data.store.book[0].title },
      { path: ['$', 'store', 'book', 0, 'author'], value: data.store.book[0].author },
      { path: ['$', 'store', 'book', 0, 'price'], value: data.store.book[0].price }
    ]);
  });

  test('union of subscript integer three keys followed by member-child-identifier', function() {
    var results = jp.nodes(data, "$.store.book[1,2,3]['title']");
    assert.deepEqual(results, [
      { path: ['$', 'store', 'book', 1, 'title'], value: data.store.book[1].title },
      { path: ['$', 'store', 'book', 2, 'title'], value: data.store.book[2].title },
      { path: ['$', 'store', 'book', 3, 'title'], value: data.store.book[3].title }
    ]);
  });

  test('union of subscript integer three keys followed by union of subscript string literal three keys', function() {
    var results = jp.nodes(data, "$.store.book[0,1,2,3]['title','author','price']");
    assert.deepEqual(results, [
      { path: ['$', 'store', 'book', 0, 'title'], value: data.store.book[0].title },
      { path: ['$', 'store', 'book', 0, 'author'], value: data.store.book[0].author },
      { path: ['$', 'store', 'book', 0, 'price'], value: data.store.book[0].price },
      { path: ['$', 'store', 'book', 1, 'title'], value: data.store.book[1].title },
      { path: ['$', 'store', 'book', 1, 'author'], value: data.store.book[1].author },
      { path: ['$', 'store', 'book', 1, 'price'], value: data.store.book[1].price },
      { path: ['$', 'store', 'book', 2, 'title'], value: data.store.book[2].title },
      { path: ['$', 'store', 'book', 2, 'author'], value: data.store.book[2].author },
      { path: ['$', 'store', 'book', 2, 'price'], value: data.store.book[2].price },
      { path: ['$', 'store', 'book', 3, 'title'], value: data.store.book[3].title },
      { path: ['$', 'store', 'book', 3, 'author'], value: data.store.book[3].author },
      { path: ['$', 'store', 'book', 3, 'price'], value: data.store.book[3].price }
    ]);
  });
  
  test('union of subscript integer four keys, including an inexistent one, followed by union of subscript string literal three keys', function() {
    var results = jp.nodes(data, "$.store.book[0,1,2,3,151]['title','author','price']");
    assert.deepEqual(results, [
      { path: ['$', 'store', 'book', 0, 'title'], value: data.store.book[0].title },
      { path: ['$', 'store', 'book', 0, 'author'], value: data.store.book[0].author },
      { path: ['$', 'store', 'book', 0, 'price'], value: data.store.book[0].price },
      { path: ['$', 'store', 'book', 1, 'title'], value: data.store.book[1].title },
      { path: ['$', 'store', 'book', 1, 'author'], value: data.store.book[1].author },
      { path: ['$', 'store', 'book', 1, 'price'], value: data.store.book[1].price },
      { path: ['$', 'store', 'book', 2, 'title'], value: data.store.book[2].title },
      { path: ['$', 'store', 'book', 2, 'author'], value: data.store.book[2].author },
      { path: ['$', 'store', 'book', 2, 'price'], value: data.store.book[2].price },
      { path: ['$', 'store', 'book', 3, 'title'], value: data.store.book[3].title },
      { path: ['$', 'store', 'book', 3, 'author'], value: data.store.book[3].author },
      { path: ['$', 'store', 'book', 3, 'price'], value: data.store.book[3].price }
    ]);
  });
  
  test('union of subscript integer three keys followed by union of subscript string literal three keys, followed by inexistent literal key', function() {
    var results = jp.nodes(data, "$.store.book[0,1,2,3]['title','author','price','fruit']");
    assert.deepEqual(results, [
      { path: ['$', 'store', 'book', 0, 'title'], value: data.store.book[0].title },
      { path: ['$', 'store', 'book', 0, 'author'], value: data.store.book[0].author },
      { path: ['$', 'store', 'book', 0, 'price'], value: data.store.book[0].price },
      { path: ['$', 'store', 'book', 1, 'title'], value: data.store.book[1].title },
      { path: ['$', 'store', 'book', 1, 'author'], value: data.store.book[1].author },
      { path: ['$', 'store', 'book', 1, 'price'], value: data.store.book[1].price },
      { path: ['$', 'store', 'book', 2, 'title'], value: data.store.book[2].title },
      { path: ['$', 'store', 'book', 2, 'author'], value: data.store.book[2].author },
      { path: ['$', 'store', 'book', 2, 'price'], value: data.store.book[2].price },
      { path: ['$', 'store', 'book', 3, 'title'], value: data.store.book[3].title },
      { path: ['$', 'store', 'book', 3, 'author'], value: data.store.book[3].author },
      { path: ['$', 'store', 'book', 3, 'price'], value: data.store.book[3].price }
    ]);
  });

  test('union of subscript 4 array slices followed by union of subscript string literal three keys', function() {
    var results = jp.nodes(data, "$.store.book[0:1,1:2,2:3,3:4]['title','author','price']");
    assert.deepEqual(results, [
      { path: ['$', 'store', 'book', 0, 'title'], value: data.store.book[0].title },
      { path: ['$', 'store', 'book', 0, 'author'], value: data.store.book[0].author },
      { path: ['$', 'store', 'book', 0, 'price'], value: data.store.book[0].price },
      { path: ['$', 'store', 'book', 1, 'title'], value: data.store.book[1].title },
      { path: ['$', 'store', 'book', 1, 'author'], value: data.store.book[1].author },
      { path: ['$', 'store', 'book', 1, 'price'], value: data.store.book[1].price },
      { path: ['$', 'store', 'book', 2, 'title'], value: data.store.book[2].title },
      { path: ['$', 'store', 'book', 2, 'author'], value: data.store.book[2].author },
      { path: ['$', 'store', 'book', 2, 'price'], value: data.store.book[2].price },
      { path: ['$', 'store', 'book', 3, 'title'], value: data.store.book[3].title },
      { path: ['$', 'store', 'book', 3, 'author'], value: data.store.book[3].author },
      { path: ['$', 'store', 'book', 3, 'price'], value: data.store.book[3].price }
    ]);
  });


  test('nested parentheses eval', function() {
    var pathExpression = '$..book[?( @.price && (@.price + 20 || false) )]'
    var results = jp.query(data, pathExpression);
    assert.deepEqual(results, data.store.book);
  });

  test('array indexes from 0 to 100', function() {
    var data = [];
    for (var i = 0; i <= 100; ++i)
      data[i] = Math.random();

    for (var i = 0; i <= 100; ++i) {
      var results = jp.query(data, '$[' + i.toString() +  ']');
      assert.deepEqual(results, [data[i]]);
    }
  });

  test('descendant subscript numeric literal', function() {
    var data = [ 0, [ 1, 2, 3 ], [ 4, 5, 6 ] ];
    var results = jp.query(data, '$..[0]');
    assert.deepEqual(results, [ 0, 1, 4 ]);
  });

  test('descendant subscript numeric literal', function() {
    var data = [ 0, 1, [ 2, 3, 4 ], [ 5, 6, 7, [ 8, 9 , 10 ] ] ];
    var results = jp.query(data, '$..[0,1]');
    assert.deepEqual(results, [ 0, 1, 2, 3, 5, 6, 8, 9 ]);
  });

  test('throws for no input', function() {
    assert.throws(function() { jp.query() }, /needs to be an object/);
  });

  test('throws for bad input', function() {
    assert.throws(function() { jp.query("string", "string") }, /needs to be an object/);
  });

  test('throws for bad input', function() {
    assert.throws(function() { jp.query({}, null) }, /we need a path/);
  });

  test('throws for bad input', function() {
    assert.throws(function() { jp.query({}, 42) }, /we need a path/);
  });

  test('union on objects', function() {
    assert.deepEqual(jp.query({a: 1, b: 2, c: null}, '$..["a","b","c","d"]'), [1, 2, null]);
  });

});

