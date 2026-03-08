import { useCallback, useEffect, useRef, useState } from "react";
import { addFile, uploadFile } from "./useApi";

export function extractFilePaths(dataTransfer: DataTransfer): string[] {
  // Try each data type that may contain file:// URIs
  for (const type of ["text/uri-list", "text/x-moz-url"]) {
    const data = dataTransfer.getData(type);
    if (data) {
      return data
        .split(/\r?\n/)
        .filter((line) => line.startsWith("file://"))
        .map((uri) => decodeURIComponent(new URL(uri).pathname));
    }
  }
  return [];
}

export function isMarkdown(name: string): boolean {
  const lower = name.toLowerCase();
  return lower.endsWith(".md") || lower.endsWith(".markdown") || lower.endsWith(".mdx");
}

function hasFiles(e: DragEvent): boolean {
  return e.dataTransfer?.types.includes("Files") ?? false;
}

export function useFileDrop(activeGroup: string): { isDragging: boolean } {
  const [isDragging, setIsDragging] = useState(false);
  const dragCounter = useRef(0);

  const handleDragEnter = useCallback((e: DragEvent) => {
    if (!hasFiles(e)) return;
    e.preventDefault();
    dragCounter.current++;
    if (dragCounter.current === 1) {
      setIsDragging(true);
    }
  }, []);

  const handleDragOver = useCallback((e: DragEvent) => {
    if (!hasFiles(e)) return;
    e.preventDefault();
  }, []);

  const handleDragLeave = useCallback((e: DragEvent) => {
    if (!hasFiles(e)) return;
    e.preventDefault();
    dragCounter.current--;
    if (dragCounter.current === 0) {
      setIsDragging(false);
    }
  }, []);

  const handleDrop = useCallback(
    async (e: DragEvent) => {
      e.preventDefault();
      dragCounter.current = 0;
      setIsDragging(false);

      if (!e.dataTransfer) return;

      // Pattern 1: file:// URI available (Firefox)
      const paths = extractFilePaths(e.dataTransfer);
      if (paths.length > 0) {
        const mdPaths = paths.filter(isMarkdown);
        await Promise.all(mdPaths.map((p) => addFile(p, activeGroup).catch(() => {})));
        return;
      }

      // Pattern 2: File objects only (Chrome/Edge) - upload content
      const fileList = e.dataTransfer.files;
      const uploads: Promise<void>[] = [];
      for (let i = 0; i < fileList.length; i++) {
        const file = fileList[i];
        if (isMarkdown(file.name)) {
          uploads.push(file.text().then((content) => uploadFile(file.name, content, activeGroup)).catch(() => {}));
        }
      }
      await Promise.all(uploads);
    },
    [activeGroup],
  );

  useEffect(() => {
    document.addEventListener("dragenter", handleDragEnter);
    document.addEventListener("dragover", handleDragOver);
    document.addEventListener("dragleave", handleDragLeave);
    document.addEventListener("drop", handleDrop);
    return () => {
      document.removeEventListener("dragenter", handleDragEnter);
      document.removeEventListener("dragover", handleDragOver);
      document.removeEventListener("dragleave", handleDragLeave);
      document.removeEventListener("drop", handleDrop);
    };
  }, [handleDragEnter, handleDragOver, handleDragLeave, handleDrop]);

  return { isDragging };
}
