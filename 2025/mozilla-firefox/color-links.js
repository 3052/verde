'use strict';

function color(elem) {
   const href = elem.getAttribute('href');
   if (href == null) {
      return;
   }
   if (href.startsWith('#')) {
      return;
   }
   if (href.startsWith('javascript:')) {
      return;
   }
   const url = new URL(href, location);
   /* override CSS important */
   elem.style.cssText = 'color: white !important';
   elem.style.textShadow = 'none';
   if (url.host == location.host) {
      elem.style.background = 'green';
   } else {
      elem.style.background = 'red';
   }
   const code = elem.querySelector('code');
   if (code != null) {
      code.style.background = elem.style.background;
   }
}

document.querySelectorAll('a').forEach(color);
