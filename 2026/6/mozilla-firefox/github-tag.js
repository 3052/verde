'use strict';

const low = location.pathname.indexOf('/releases/');

if (low >= 0) {
   const root = location.pathname.substring(0, low);
   const high = location.pathname.lastIndexOf('/');
   if (high >= 0) {
      const tag = location.pathname.substring(high);
      location.pathname = root + '/compare' + tag + '...main';
   }
}
