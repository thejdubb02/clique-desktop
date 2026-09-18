"use strict";

const fs = require("fs");
const path = require("path");

const src = fs.readFileSync(path.join(__dirname, "..", "caption.js"), "utf8");

function extractCssColorToInt(source) {
  const marker = "window.cssColorToInt = ";
  const i = source.indexOf(marker);
  if (i < 0) {
    throw new Error("caption.js does not assign window.cssColorToInt");
  }
  const start = source.indexOf("function", i);
  if (start < 0 || start > i + marker.length + 16) {
    throw new Error("cssColorToInt is not a function");
  }
  const brace = source.indexOf("{", start);
  let depth = 0;
  for (let j = brace; j < source.length; j++) {
    const c = source[j];
    if (c === "{") depth++;
    else if (c === "}") {
      depth--;
      if (depth === 0) {
        return source.slice(start, j + 1);
      }
    }
  }
  throw new Error("cssColorToInt function is not closed");
}

const cssColorToInt = eval("(" + extractCssColorToInt(src) + ")");

const cases = [
  ["#abc", 0xAABBCC],
  ["#1e1e1e", 0x1E1E1E],
  [" rgb(37, 37, 38) ", 0x252526],
  ["rgba(1, 2, 3, 0.5)", 0x010203],
  ["garbage", null],
];

let failed = 0;
for (const [input, want] of cases) {
  const got = cssColorToInt(input);
  if (got !== want) {
    console.error("cssColorToInt(" + JSON.stringify(input) + ") = " + got + ", want " + want);
    failed++;
  }
}
if (failed) process.exit(1);
console.log("caption colours: " + cases.length + " ok");
