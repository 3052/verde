# openstyles-stylus

- <https://addons.mozilla.org/firefox/addon/styl-us>
- <https://github.com/openstyles/stylus>

## install

> Firefox on Android doesn't provide an option to revert extensions, it's not our fault.

this is 100% your fault. it wasnt Firefox or uBlock Origin or something else that broke, it was Stylus.

> This bug report is closed because it's fixed in the source code, that's the standard behavior for bug trackers. The workarounds are posted above.

since developer doesnt seem interested in helping users impacted by this, here is an actual solution:

1. <https://addons.mozilla.org/firefox/downloads/file/4338993/styl_us-2.3.9.xpi>
2. chrome://geckoview/content/config.xhtml
3. extensions.update.enabled = false
4. settings
5. about firefox
6. tap logo five times
7. navigate up
8. install extension from file
