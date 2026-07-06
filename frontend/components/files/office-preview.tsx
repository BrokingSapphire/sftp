"use client";

import { useEffect, useState } from "react";
import { Loader2, FileWarning } from "lucide-react";
import { filesApi } from "@/lib/endpoints";

function Center({ children }: { children: React.ReactNode }) {
  return <div className="flex h-full w-full items-center justify-center">{children}</div>;
}
function Loading() {
  return <Center><Loader2 className="animate-spin text-white/70" size={28} /></Center>;
}
function Failed({ label }: { label: string }) {
  return (
    <Center>
      <div className="flex flex-col items-center gap-2 text-white/60">
        <FileWarning size={36} />
        <p className="text-sm">{label}</p>
      </div>
    </Center>
  );
}

/* ── Spreadsheets: xlsx / xls / xlsm / ods (SheetJS) ── */
export function SpreadsheetPreview({ fileId }: { fileId: string }) {
  const [sheets, setSheets] = useState<{ name: string; rows: string[][] }[] | null>(null);
  const [active, setActive] = useState(0);
  const [err, setErr] = useState(false);

  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const [{ read, utils }, buf] = await Promise.all([import("xlsx"), filesApi.fetchBinary(fileId)]);
        const wb = read(buf, { type: "array" });
        const out = wb.SheetNames.map((name) => ({
          name,
          rows: (utils.sheet_to_json(wb.Sheets[name], { header: 1, blankrows: false, defval: "" }) as string[][])
            .slice(0, 2000)
            .map((r) => r.map((c) => String(c ?? "")).slice(0, 60)),
        }));
        if (alive) setSheets(out);
      } catch { if (alive) setErr(true); }
    })();
    return () => { alive = false; };
  }, [fileId]);

  if (err) return <Failed label="Could not read spreadsheet." />;
  if (!sheets) return <Loading />;
  const sheet = sheets[active];

  return (
    <div className="flex h-full w-full max-w-6xl flex-col overflow-hidden rounded-lg bg-white shadow-2xl dark:bg-[#12191b]">
      <div className="flex-1 overflow-auto">
        <table className="w-full border-collapse text-sm text-zinc-900 dark:text-zinc-100">
          <tbody>
            {sheet.rows.map((cells, i) => (
              <tr key={i} className={i === 0 ? "sticky top-0 z-10 bg-surface-2 font-semibold" : "odd:bg-black/[0.015] dark:odd:bg-white/[0.02]"}>
                <td className="select-none border border-black/5 bg-black/[0.03] px-2 py-1 text-center text-[10px] text-zinc-400 dark:border-white/5 dark:bg-white/5">{i + 1}</td>
                {cells.map((c, j) => (
                  <td key={j} className="whitespace-nowrap border border-black/5 px-3 py-1.5 dark:border-white/5">{c}</td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {sheets.length > 1 && (
        <div className="flex gap-1 overflow-x-auto border-t border-black/10 bg-surface-2 px-2 py-1.5 dark:border-white/10">
          {sheets.map((s, i) => (
            <button
              key={s.name}
              onClick={() => setActive(i)}
              className={`shrink-0 rounded px-3 py-1 text-xs font-medium ${i === active ? "bg-primary text-primary-foreground" : "text-muted hover:bg-black/5 dark:hover:bg-white/10"}`}
            >
              {s.name}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

/* ── Word: docx (mammoth → HTML) ── */
export function DocxPreview({ fileId }: { fileId: string }) {
  const [html, setHtml] = useState<string | null>(null);
  const [err, setErr] = useState(false);

  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const [{ convertToHtml }, buf] = await Promise.all([import("mammoth"), filesApi.fetchBinary(fileId)]);
        const res = await convertToHtml({ arrayBuffer: buf });
        if (alive) setHtml(res.value || "<p>(empty document)</p>");
      } catch { if (alive) setErr(true); }
    })();
    return () => { alive = false; };
  }, [fileId]);

  if (err) return <Failed label="Could not render document. Legacy .doc is not supported — download it." />;
  if (html === null) return <Loading />;

  return (
    <div className="h-full w-full max-w-3xl overflow-auto rounded-lg bg-white shadow-2xl">
      <div className="office-doc mx-auto max-w-[46rem] px-12 py-14 text-zinc-900" dangerouslySetInnerHTML={{ __html: html }} />
    </div>
  );
}

/* ── PowerPoint: pptx (JSZip → slide text extraction) ── */
export function PptxPreview({ fileId }: { fileId: string }) {
  const [slides, setSlides] = useState<string[][] | null>(null);
  const [err, setErr] = useState(false);

  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const [JSZip, buf] = await Promise.all([import("jszip").then((m) => m.default), filesApi.fetchBinary(fileId)]);
        const zip = await JSZip.loadAsync(buf);
        const names = Object.keys(zip.files)
          .filter((n) => /^ppt\/slides\/slide\d+\.xml$/.test(n))
          .sort((a, b) => Number(a.match(/\d+/)![0]) - Number(b.match(/\d+/)![0]));
        const parser = new DOMParser();
        const out: string[][] = [];
        for (const n of names) {
          const xml = await zip.files[n].async("text");
          const doc = parser.parseFromString(xml, "application/xml");
          const runs = Array.from(doc.getElementsByTagName("a:t")).map((t) => t.textContent ?? "").filter(Boolean);
          out.push(runs);
        }
        if (alive) setSlides(out);
      } catch { if (alive) setErr(true); }
    })();
    return () => { alive = false; };
  }, [fileId]);

  if (err) return <Failed label="Could not read presentation. Legacy .ppt is not supported — download it." />;
  if (!slides) return <Loading />;

  return (
    <div className="h-full w-full max-w-4xl space-y-4 overflow-auto p-2">
      {slides.map((runs, i) => (
        <div key={i} className="relative aspect-[16/9] w-full overflow-auto rounded-lg border border-white/10 bg-white p-8 text-zinc-900 shadow-xl">
          <span className="absolute right-3 top-3 font-mono text-[10px] text-zinc-400">Slide {i + 1}</span>
          {runs.length === 0 ? (
            <p className="text-sm text-zinc-400">(no text on this slide)</p>
          ) : (
            <>
              {runs[0] && <h3 className="mb-3 text-xl font-semibold">{runs[0]}</h3>}
              <ul className="space-y-1.5 text-sm leading-relaxed">
                {runs.slice(1).map((r, j) => <li key={j}>{r}</li>)}
              </ul>
            </>
          )}
        </div>
      ))}
      {slides.length === 0 && <Failed label="No slides found." />}
    </div>
  );
}
