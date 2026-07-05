import {
  FileText, FileImage, FileVideo, FileAudio, FileArchive,
  FileCode, FileSpreadsheet, File as FileIcon,
} from "lucide-react";

const map: Record<string, { icon: React.ElementType; color: string }> = {
  pdf: { icon: FileText, color: "text-red-500" },
  doc: { icon: FileText, color: "text-blue-500" },
  docx: { icon: FileText, color: "text-blue-500" },
  txt: { icon: FileText, color: "text-slate-500" },
  md: { icon: FileText, color: "text-slate-500" },
  csv: { icon: FileSpreadsheet, color: "text-green-600" },
  xls: { icon: FileSpreadsheet, color: "text-green-600" },
  xlsx: { icon: FileSpreadsheet, color: "text-green-600" },
  png: { icon: FileImage, color: "text-purple-500" },
  jpg: { icon: FileImage, color: "text-purple-500" },
  jpeg: { icon: FileImage, color: "text-purple-500" },
  gif: { icon: FileImage, color: "text-purple-500" },
  svg: { icon: FileImage, color: "text-purple-500" },
  webp: { icon: FileImage, color: "text-purple-500" },
  mp4: { icon: FileVideo, color: "text-pink-500" },
  mov: { icon: FileVideo, color: "text-pink-500" },
  webm: { icon: FileVideo, color: "text-pink-500" },
  mp3: { icon: FileAudio, color: "text-amber-500" },
  wav: { icon: FileAudio, color: "text-amber-500" },
  zip: { icon: FileArchive, color: "text-yellow-600" },
  tar: { icon: FileArchive, color: "text-yellow-600" },
  gz: { icon: FileArchive, color: "text-yellow-600" },
  rar: { icon: FileArchive, color: "text-yellow-600" },
  json: { icon: FileCode, color: "text-emerald-500" },
  xml: { icon: FileCode, color: "text-emerald-500" },
  js: { icon: FileCode, color: "text-emerald-500" },
  ts: { icon: FileCode, color: "text-emerald-500" },
  go: { icon: FileCode, color: "text-emerald-500" },
  py: { icon: FileCode, color: "text-emerald-500" },
};

export function fileIcon(ext: string, size = 20) {
  const entry = map[ext?.toLowerCase()] ?? { icon: FileIcon, color: "text-muted" };
  const Icon = entry.icon;
  return <Icon size={size} className={entry.color} />;
}
