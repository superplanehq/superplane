var assert = require('assert')

var detect = require('../src/detect-mac');

describe("Mac os tests", function () {
  it ('detects chrome', function (done) {
    detect(function (err, res) {
      assert.equal(res.isChrome, true);
      assert.equal(res.isChromium, false);
      assert.equal(res.isWebkit, true);
      assert.equal(res.isBlink, true);
      assert.equal(res.commonName, 'chrome');
      assert.equal(res.identity, 'com.google.chrome');
      done(err);
    })
  })
})