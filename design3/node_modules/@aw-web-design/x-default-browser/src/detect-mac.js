const defaultBrowserId = import('default-browser-id');

module.exports = async function (callback) {
  const id = await defaultBrowserId.then(res => res.default());

  if (id) {
    const idLC = id.toLowerCase();

    /*
              Safari         com.apple.Safari
              Google Chrome  com.google.chrome
              Opera          com.operasoftware.Opera
              Firefox        org.mozilla.firefox
           */

    const out = {
      isIE: false,
      isSafari: idLC.indexOf('safari') > -1,
      isFirefox: idLC.indexOf('firefox') > -1,
      isChrome: idLC.indexOf('google') > -1,
      isChromium: idLC.indexOf('chromium') > -1, // untested
      isOpera: idLC.indexOf('opera') > -1,
      identity: id,
    };
    out.isBlink = out.isChrome || out.isChromium || out.isOpera;
    out.isWebkit = out.isSafari || out.isBlink;
    out.commonName = require('./common-name')(out);

    callback(null, out);
  } else {
    callback('Unable to retrieve default browser');
  }
};
