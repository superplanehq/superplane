/**
 * Copyright (c) 2021 ESHA Research
 * Dual licensed under the MIT and GPL licenses:
 *   http://www.opensource.org/licenses/mit-license.php
 *   http://www.gnu.org/licenses/gpl.html
 *
 * Creates a store area that uses a single cookie as the backing storage.
 * This gives you the store API for a specific 'store' cookie that your backend
 * can access too. It could definitely use more testing.
 *
 * Status: BETA - unsupported, useful, needs testing
 */
;(function(window, document, store, _) {

    var C = _.cookie = {// and that's good enough for me
        name: 'store',
        maxAge: 60*60*24*365*10,
        suffix: ';path=/;sameSite=strict',
        encode: function(state) {
            return encodeURIComponent(JSON.stringify(state));
        },
        decode: function(state) {
            return JSON.parse(decodeURIComponent(state));
        }
    };
    C.all = function() {
        return C.read(C.name) || {};
    };
    C.read = function(name) {
        var match = document.cookie.match(new RegExp("(^| )"+(name||C.name)+"=([^;]+)"));
        return match ? C.decode(match[2]) : null;
    };
    C.write = function(state, name) {
        document.cookie = (name||C.name)+"="+C.encode(state)+";max-age="+C.maxAge+C.suffix;
    };
    C.remove = function(name) {
        document.cookie = (name||C.name)+"=;expires=Thu, 01 Jan 1970 00:00:01 GMT"+C.suffix;
    };
    C.area = {
        key: function(i) {
            var c = 0,
                state = C.all();
            for (var k in state) {
                if (state.hasOwnProperty(k) && i === c++) {
                    return k;
                }
            }
        },
        setItem: function(k, v) {
            var state = C.all();
            state[k] = v;
            C.write(state);
        },
        getItem: function(k) {
            var state = C.all();
            return state.hasOwnProperty(k) ? state[k] : null;
        },
        has: function(k) {
            return C.all().hasOwnProperty(k);
        },
        removeItem: function(k) {
            var state = C.read(C.name);
            delete state[k];
            C.write(state);
        },
        clear: C.remove
    };
    Object.defineProperty(C.area, "length", {
        get: function() {
            var ln = 0,
                state = C.all();
            for (var k in state) {
                if (state.hasOwnProperty(k)) {
                    ln++;
                }
            }
            return ln;
        }
    });

    // create the store api for this storage
    store.area("cookie", C.area);

})(window, document, window.store, window.store._);
