javascript: (function () {
    'use strict';

    /* 1. Create a TreeWalker to safely scan only text nodes */
    const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT, null, false);
    const nodes = [];
    let node;

    /* 2. Find all valid text nodes containing 4K or UHD as whole words */
    while ((node = walker.nextNode())) {
        const parentTag = node.parentNode.tagName;
        
        /* Skip code blocks, text boxes, and already highlighted text */
        if (['SCRIPT', 'STYLE', 'NOSCRIPT', 'TEXTAREA', 'MARK'].includes(parentTag)) {
            continue;
        }
        
        /* Word boundaries prevent matching partial words like 14K */
        if (/\b(4K|UHD)\b/gi.test(node.nodeValue)) {
            nodes.push(node);
        }
    }

    /* 3. Process and highlight the matched nodes */
    nodes.forEach(n => {
        const frag = document.createElement('span');

        /* Safely escape existing HTML characters */
        const safeText = n.nodeValue
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;');

        /* Inject the mark tag with your requested CSS styles */
        frag.innerHTML = safeText.replace(
            /\b(4K|UHD)\b/gi,
            '<mark style="background: red !important; color: white !important; text-shadow: none !important; font-weight: bold !important; padding: 0 3px !important; border-radius: 3px !important;">$1</mark>'
        );

        /* Replace original text with highlighted HTML elements */
        n.replaceWith(...frag.childNodes);
    });
})();
