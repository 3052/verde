# Firefox

- https://archive.mozilla.org/pub/firefox/releases/
- https://archive.mozilla.org/pub/mobile/releases/68.11.0/

## bookmarklet

~~~js
javascript: {
   const n = 10;
   console.log(n);
} void 0;
~~~

## command options

Bypass Profile Manager and launch application with a profile:

~~~
firefox -no-remote -P hello
~~~

Start with Profile Manager:

~~~
firefox -P
~~~

Make sure to not name the default profile, as then script will require:

~~~
firefox -P hello
~~~

If you did this already, just rename to a blank name.

<https://developer.mozilla.org/Mozilla/Command_Line_Options>

## policies.json

~~~
C:\Program Files\Mozilla Firefox\distribution\policies.json
~~~

## user agent

~~~
general.useragent.override
~~~

## widevine

~~~
archive.mozilla.org/pub/firefox/releases/128.5.0esr/win64/en-US/
4.10.2830.0
~~~
