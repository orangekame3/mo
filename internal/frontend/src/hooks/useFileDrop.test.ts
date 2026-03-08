import { describe, it, expect } from "vitest";
import { extractFilePaths, isMarkdown } from "./useFileDrop";

describe("extractFilePaths", () => {
  function makeDT(data: Record<string, string>): DataTransfer {
    return {
      getData: (type: string) => data[type] ?? "",
    } as unknown as DataTransfer;
  }

  it("extracts paths from text/uri-list", () => {
    const dt = makeDT({
      "text/uri-list": "file:///home/user/docs/a.md\nfile:///home/user/docs/b.md",
    });
    expect(extractFilePaths(dt)).toEqual(["/home/user/docs/a.md", "/home/user/docs/b.md"]);
  });

  it("extracts paths from text/x-moz-url", () => {
    const dt = makeDT({
      "text/x-moz-url": "file:///tmp/test.md\ntest.md",
    });
    expect(extractFilePaths(dt)).toEqual(["/tmp/test.md"]);
  });

  it("prefers text/uri-list over text/x-moz-url", () => {
    const dt = makeDT({
      "text/uri-list": "file:///a.md",
      "text/x-moz-url": "file:///b.md",
    });
    expect(extractFilePaths(dt)).toEqual(["/a.md"]);
  });

  it("filters out non-file URIs", () => {
    const dt = makeDT({
      "text/uri-list": "https://example.com\nfile:///a.md",
    });
    expect(extractFilePaths(dt)).toEqual(["/a.md"]);
  });

  it("returns empty array when no data", () => {
    const dt = makeDT({});
    expect(extractFilePaths(dt)).toEqual([]);
  });

  it("decodes percent-encoded paths", () => {
    const dt = makeDT({
      "text/uri-list": "file:///home/user/my%20docs/file%20name.md",
    });
    expect(extractFilePaths(dt)).toEqual(["/home/user/my docs/file name.md"]);
  });
});

describe("isMarkdown", () => {
  it("accepts .md files", () => {
    expect(isMarkdown("readme.md")).toBe(true);
  });

  it("accepts .markdown files", () => {
    expect(isMarkdown("doc.markdown")).toBe(true);
  });

  it("accepts .mdx files", () => {
    expect(isMarkdown("page.mdx")).toBe(true);
  });

  it("is case insensitive", () => {
    expect(isMarkdown("README.MD")).toBe(true);
    expect(isMarkdown("Doc.Markdown")).toBe(true);
    expect(isMarkdown("Page.MDX")).toBe(true);
  });

  it("rejects non-markdown files", () => {
    expect(isMarkdown("script.js")).toBe(false);
    expect(isMarkdown("style.css")).toBe(false);
    expect(isMarkdown("image.png")).toBe(false);
    expect(isMarkdown("data.json")).toBe(false);
  });
});
