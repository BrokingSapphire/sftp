// Helpers to read a folder into a flat list of files with relative paths, for
// prompt-free folder uploads. Two sources: the File System Access API
// (Chromium directory picker) and drag-dropped directory entries (all browsers,
// including Firefox/Zen — no "trust this site" confirm).

export interface FolderEntry {
  file: File;
  relPath: string;
}

/** Recursively read a File System Access API directory handle. */
export async function walkDir(
  dir: FileSystemDirectoryHandle,
  prefix: string,
  out: FolderEntry[],
): Promise<void> {
  // @ts-expect-error entries() is async-iterable on directory handles
  for await (const [name, handle] of dir.entries()) {
    const path = `${prefix}/${name}`;
    if (handle.kind === "file") {
      out.push({ file: await handle.getFile(), relPath: path });
    } else {
      await walkDir(handle, path, out);
    }
  }
}

/** Recursively read a drag-dropped directory entry (webkitGetAsEntry). */
export async function readDropEntry(
  entry: FileSystemEntry,
  prefix: string,
  out: FolderEntry[],
): Promise<void> {
  const path = prefix ? `${prefix}/${entry.name}` : entry.name;
  if (entry.isFile) {
    const file = await new Promise<File>((res, rej) => (entry as FileSystemFileEntry).file(res, rej));
    out.push({ file, relPath: path });
  } else if (entry.isDirectory) {
    const reader = (entry as FileSystemDirectoryEntry).createReader();
    const children: FileSystemEntry[] = await new Promise((res) => {
      const acc: FileSystemEntry[] = [];
      const read = () =>
        reader.readEntries((batch) => {
          if (batch.length === 0) return res(acc);
          acc.push(...batch);
          read();
        });
      read();
    });
    for (const child of children) await readDropEntry(child, path, out);
  }
}
