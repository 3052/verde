'use strict';

const prefs = {
   // always ask me where to save files
   'browser.download.folderList': 0,
   // disable sponsored results
   'browser.newtabpage.activity-stream.showSponsoredTopSites': false,
   // disable new tab page
   'browser.newtabpage.enabled': false,
   // do not provide search suggestions
   'browser.search.suggest.enabled': false,
   // disable default browser nag
   'browser.shell.checkDefaultBrowser': false,
   // show windows and tabs from last time
   'browser.startup.page': 3,
   // disable delay hiding mute tab
   'browser.tabs.delayHidingAudioPlayingIconMS': 0,
   // jumplist setting
   'browser.taskbar.lists.enabled': false,
   // disable URL autocomplete
   'browser.urlbar.autoFill': false,
   // switch to tab
   'browser.urlbar.suggest.openpage': false,
   // browser console
   'devtools.chrome.enabled': true,
   // disable notifications
   'dom.webnotifications.enabled': false,
   // fuck you pocket piece of shit
   'extensions.pocket.enabled': false,
   // fix default shitty jerky ass scrolling
   'general.smoothScroll.mouseWheel.durationMaxMS': 400,
   'general.smoothScroll.mouseWheel.durationMinMS': 200,
   // allow autoplay
   'media.autoplay.default': 0,
   // allow autoplay
   'media.block-autoplay-until-in-foreground': false,
   // youtube
   'network.cookie.cookieBehavior': 1,
   // youtube
   'privacy.trackingprotection.pbmode.enabled': false,
   // remember passwords
   'signon.rememberSignons': false,
   // developer.mozilla.org/docs/Web/CSS/@media/prefers-reduced-motion
   'ui.prefersReducedMotion': 1,
};

Services.prefs.getChildList('').forEach(p => Services.prefs.clearUserPref(p));

for (const [key, value] of Object.entries(prefs)) {
   if (Services.prefs.getPrefType(key) == Services.prefs.PREF_INVALID) {
      console.log('INVALID', key);
   }
   switch (typeof value) {
   case 'boolean':
      Services.prefs.setBoolPref(key, value);
      break;
   case 'number':
      Services.prefs.setIntPref(key, value);
   }
   if (!Services.prefs.prefHasUserValue(key)) {
      console.log('DEFAULT', key);
   }
}
