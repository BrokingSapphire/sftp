import {
  FileText, FileImage, FileVideo, FileAudio, FileArchive, FileCode,
  FileSpreadsheet, FileJson, FileType2, FileCog, Database, FileTerminal,
  Presentation, Braces, Sheet, FileLock2, Binary, BookText, File as FileIcon,
} from "lucide-react";

type Entry = { icon: React.ElementType; color: string };

// Best-effort, per-extension icon + brand-ish colour. Grouped by category.
const MAP: Record<string, Entry> = {
  // Documents
  pdf: { icon: FileText, color: "text-red-500" },
  doc: { icon: FileText, color: "text-blue-600" },
  docx: { icon: FileText, color: "text-blue-600" },
  rtf: { icon: FileText, color: "text-blue-500" },
  odt: { icon: FileText, color: "text-blue-500" },
  txt: { icon: FileText, color: "text-zinc-500" },
  md: { icon: BookText, color: "text-zinc-600" },
  markdown: { icon: BookText, color: "text-zinc-600" },
  epub: { icon: BookText, color: "text-emerald-600" },

  // Spreadsheets
  csv: { icon: Sheet, color: "text-green-600" },
  tsv: { icon: Sheet, color: "text-green-600" },
  xls: { icon: FileSpreadsheet, color: "text-green-700" },
  xlsx: { icon: FileSpreadsheet, color: "text-green-700" },
  xlsm: { icon: FileSpreadsheet, color: "text-green-700" },
  ods: { icon: FileSpreadsheet, color: "text-green-600" },

  // Presentations
  ppt: { icon: Presentation, color: "text-orange-600" },
  pptx: { icon: Presentation, color: "text-orange-600" },
  odp: { icon: Presentation, color: "text-orange-500" },

  // Images
  png: { icon: FileImage, color: "text-purple-500" },
  jpg: { icon: FileImage, color: "text-purple-500" },
  jpeg: { icon: FileImage, color: "text-purple-500" },
  gif: { icon: FileImage, color: "text-purple-500" },
  webp: { icon: FileImage, color: "text-purple-500" },
  bmp: { icon: FileImage, color: "text-purple-500" },
  tiff: { icon: FileImage, color: "text-purple-500" },
  svg: { icon: FileImage, color: "text-pink-500" },
  ico: { icon: FileImage, color: "text-purple-400" },
  avif: { icon: FileImage, color: "text-purple-500" },
  heic: { icon: FileImage, color: "text-purple-500" },
  psd: { icon: FileImage, color: "text-blue-500" },
  ai: { icon: FileImage, color: "text-orange-500" },
  fig: { icon: FileImage, color: "text-pink-600" },

  // Video
  mp4: { icon: FileVideo, color: "text-rose-500" },
  mov: { icon: FileVideo, color: "text-rose-500" },
  webm: { icon: FileVideo, color: "text-rose-500" },
  mkv: { icon: FileVideo, color: "text-rose-500" },
  avi: { icon: FileVideo, color: "text-rose-500" },
  flv: { icon: FileVideo, color: "text-rose-400" },

  // Audio
  mp3: { icon: FileAudio, color: "text-amber-500" },
  wav: { icon: FileAudio, color: "text-amber-500" },
  flac: { icon: FileAudio, color: "text-amber-600" },
  ogg: { icon: FileAudio, color: "text-amber-500" },
  m4a: { icon: FileAudio, color: "text-amber-500" },
  aac: { icon: FileAudio, color: "text-amber-500" },

  // Archives
  zip: { icon: FileArchive, color: "text-yellow-600" },
  rar: { icon: FileArchive, color: "text-yellow-600" },
  "7z": { icon: FileArchive, color: "text-yellow-600" },
  tar: { icon: FileArchive, color: "text-yellow-700" },
  gz: { icon: FileArchive, color: "text-yellow-700" },
  bz2: { icon: FileArchive, color: "text-yellow-700" },
  xz: { icon: FileArchive, color: "text-yellow-700" },

  // Data / config
  json: { icon: FileJson, color: "text-emerald-500" },
  jsonc: { icon: FileJson, color: "text-emerald-500" },
  yaml: { icon: FileCog, color: "text-teal-500" },
  yml: { icon: FileCog, color: "text-teal-500" },
  toml: { icon: FileCog, color: "text-teal-600" },
  ini: { icon: FileCog, color: "text-zinc-500" },
  conf: { icon: FileCog, color: "text-zinc-500" },
  env: { icon: FileLock2, color: "text-amber-600" },
  xml: { icon: FileCode, color: "text-orange-500" },
  sql: { icon: Database, color: "text-sky-600" },
  db: { icon: Database, color: "text-sky-700" },
  sqlite: { icon: Database, color: "text-sky-700" },

  // Code — web
  html: { icon: FileCode, color: "text-orange-600" },
  htm: { icon: FileCode, color: "text-orange-600" },
  css: { icon: FileCode, color: "text-sky-500" },
  scss: { icon: FileCode, color: "text-pink-500" },
  js: { icon: FileCode, color: "text-yellow-500" },
  mjs: { icon: FileCode, color: "text-yellow-500" },
  jsx: { icon: FileCode, color: "text-cyan-500" },
  ts: { icon: FileCode, color: "text-blue-500" },
  tsx: { icon: FileCode, color: "text-cyan-600" },
  vue: { icon: FileCode, color: "text-emerald-500" },

  // Code — languages
  go: { icon: FileCode, color: "text-cyan-600" },
  py: { icon: FileCode, color: "text-blue-400" },
  rb: { icon: FileCode, color: "text-red-500" },
  php: { icon: FileCode, color: "text-indigo-500" },
  java: { icon: FileCode, color: "text-red-600" },
  kt: { icon: FileCode, color: "text-violet-500" },
  c: { icon: FileCode, color: "text-blue-500" },
  h: { icon: FileCode, color: "text-blue-400" },
  cpp: { icon: FileCode, color: "text-blue-600" },
  cs: { icon: FileCode, color: "text-green-600" },
  rs: { icon: FileCode, color: "text-orange-700" },
  swift: { icon: FileCode, color: "text-orange-500" },
  dart: { icon: FileCode, color: "text-sky-500" },
  lua: { icon: Braces, color: "text-indigo-400" },

  // Shell / binary / misc
  sh: { icon: FileTerminal, color: "text-zinc-600" },
  bash: { icon: FileTerminal, color: "text-zinc-600" },
  zsh: { icon: FileTerminal, color: "text-zinc-600" },
  ps1: { icon: FileTerminal, color: "text-blue-500" },
  exe: { icon: Binary, color: "text-zinc-500" },
  bin: { icon: Binary, color: "text-zinc-500" },
  dmg: { icon: Binary, color: "text-zinc-500" },
  iso: { icon: FileArchive, color: "text-zinc-500" },
  ttf: { icon: FileType2, color: "text-fuchsia-500" },
  otf: { icon: FileType2, color: "text-fuchsia-500" },
  woff: { icon: FileType2, color: "text-fuchsia-400" },
  woff2: { icon: FileType2, color: "text-fuchsia-400" },
};

export function fileIcon(ext: string, size = 20) {
  const e = ext?.toLowerCase() ?? "";
  const entry = MAP[e] ?? { icon: FileIcon, color: "text-muted" };
  const Icon = entry.icon;
  return <Icon size={size} className={entry.color} />;
}
