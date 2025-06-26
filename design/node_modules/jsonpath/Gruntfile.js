var Browserify = require('browserify');
var bresolve = require('browser-resolve');
patchResolve();

module.exports = function (grunt) {
    grunt.initConfig({
        pkg: grunt.file.readJSON('package.json'),
        outputFolder: ".",

        browserify: {
            main: {
                src: ['index.js'],
                dest: '<%= outputFolder %>/<%= pkg.name %>.js',
                options: {
                    browserifyOptions: { standalone: '<%= pkg.name %>' },
                    banner: '/*! <%= pkg.name %> <%= pkg.version %> */\n',
                    alias: {
                        "jsonpath": "./index.js"
                    },
                    require: [
                        /**
                         * When running in Node, we require('./aesprim') and that module takes care of monkey-patching esprima
                         * using resolve, path finding, etc...
                         * Anyways, Browserify doesn't support "resolve", so we need to trick the module. We'll actually be
                         * returning a verbatim, non-modified "esprima" when the code runs require('./aesprim').
                         * That is ok because we will modify the "esprima" source code right after the bundle process, via
                         * the postBundleCB callback.
                         */
                        ["esprima", {expose: "./aesprim"}]
                    ],
                    ignore: [
                        'file',
                        'system',
                        'source-map',
                        'estraverse',
                        'escodegen',
                        'underscore',
                        'reflect',
                        'JSONSelect',
                        './lib/aesprim.js'
                        //'assert' //can't remove because of lib/index.js,
                    ],
                    postBundleCB: function(err, src, next) {
                        /**
                         * This is ugly, but we need to make "esprima" understand '@' as a valid character.
                         * It's either this or bundle a copy of the library with those few bytes of changes.
                         */
                        src = src.toString("utf8").replace(/(function isIdentifierStart\(ch\) {\s+return)/m, '$1 (ch == 0x40) || ');
                        next(err, new Buffer(src, "utf8"));
                    }
                }
            }
        },

        uglify: {
            options: {
                banner: '/*! <%= pkg.name %> <%= pkg.version %> */\n'
            },
            build: {
                src: '<%= outputFolder %>/<%= pkg.name %>.js',
                dest: '<%= outputFolder %>/<%= pkg.name %>.min.js'
            }
        }

    });

    grunt.loadNpmTasks('grunt-browserify');
    grunt.loadNpmTasks('grunt-contrib-uglify')
    grunt.registerTask('default', ['browserify', 'uglify']);

};

function patchResolve() {
  var _createDeps = Browserify.prototype._createDeps;
  Browserify.prototype._createDeps = function() {
    var returnValue = _createDeps.apply(this, arguments);
    this._bresolve = function(id, opts, cb) {
      opts.browser = 'alias';
      return bresolve(id, opts, cb);
    };
    return returnValue;
  }
}
