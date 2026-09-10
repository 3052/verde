const STORAGE_KEY = "longpress-visited";
const VISITED_COLOR = "hsl(270 100% 50%)";

function readVisited() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);

    if (raw === null) {
      return {};
    }

    return JSON.parse(raw);
  } catch (error) {
    console.error("Mark visited: could not read stored URLs", error);
    return {};
  }
}

function paintVisited(link) {
  link.style.setProperty("color", VISITED_COLOR, "important");
}

function saveVisited(url) {
  const visited = readVisited();
  visited[url] = true;

  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(visited));
  } catch (error) {
    console.error("Mark visited: could not save URL", error);
  }
}

document.addEventListener("contextmenu", (event) => {
  if (!(event.target instanceof Element)) {
    return;
  }

  const link = event.target.closest("a[href]");

  if (link === null) {
    return;
  }

  paintVisited(link);
  saveVisited(link.href);
}, true);

const visitedUrls = new Set(Object.keys(readVisited()));

for (const link of document.querySelectorAll("a[href]")) {
  if (visitedUrls.has(link.href)) {
    paintVisited(link);
  }
}
