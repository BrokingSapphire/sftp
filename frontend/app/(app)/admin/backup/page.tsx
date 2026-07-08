"use client";

import { useState } from "react";
import { toast } from "sonner";
import { HardDriveDownload, HardDriveUpload, Loader2, ShieldAlert, Database, RotateCcw, Lock } from "lucide-react";
import { backupApi, type BackupStatus, type BackupResult } from "@/lib/endpoints";
import { ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { PageHeader } from "@/components/files/file-list";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useDialogs } from "@/components/ui/dialogs";
import { EmptyState } from "@/components/ui/empty-state";
import { formatBytes, timeAgo } from "@/lib/utils";

export default function BackupPage() {
  const { user } = useAuth();
  const { confirm } = useDialogs();
  const [path, setPath] = useState("/backups");
  const [status, setStatus] = useState<BackupStatus | null>(null);
  const [result, setResult] = useState<BackupResult | null>(null);
  const [busy, setBusy] = useState<"status" | "run" | "restore" | null>(null);

  if (user?.role !== "super_admin") {
    return (
      <div className="mx-auto max-w-2xl">
        <PageHeader title="Backup & Restore" subtitle="Super-admin only" />
        <EmptyState icon={Lock} title="Restricted area" subtitle="Only a super admin can back up or restore the platform. Nice try, though." />
      </div>
    );
  }

  function err(e: unknown, fallback: string) {
    toast.error(e instanceof ApiError ? e.message : fallback);
  }

  async function check() {
    setBusy("status");
    try { setStatus(await backupApi.status(path)); }
    catch (e) { err(e, "Could not read the target"); setStatus(null); }
    finally { setBusy(null); }
  }
  async function run() {
    setBusy("run"); setResult(null);
    try {
      const r = await backupApi.run(path);
      setResult(r);
      toast.success(r.mode === "none" ? "Already up to date" : `${r.mode === "full" ? "Full" : "Incremental"} backup complete`);
      check();
    } catch (e) { err(e, "Backup failed"); }
    finally { setBusy(null); }
  }
  async function restore() {
    if (!(await confirm({ title: "Restore from backup", message: "Restore ALL users' files from this target? Missing files are recreated; existing ones are left untouched.", confirmLabel: "Restore" }))) return;
    setBusy("restore");
    try {
      const r = await backupApi.restore(path);
      toast.success(`Restored ${r.restored} file${r.restored === 1 ? "" : "s"} (${r.skipped} already present)`);
    } catch (e) { err(e, "Restore failed"); }
    finally { setBusy(null); }
  }

  return (
    <div className="mx-auto max-w-3xl space-y-4">
      <PageHeader title="Backup & Restore" subtitle="Encrypted, incremental backups of every user's drive to a removable disk" />

      <div className="flex items-start gap-2 rounded-xl border border-primary/30 bg-primary/5 px-4 py-3 text-sm">
        <ShieldAlert size={18} className="mt-0.5 shrink-0 text-primary" />
        <p className="text-muted">
          Backups are <strong className="text-foreground">AES-256 encrypted</strong> binary archives. The first run to a
          fresh disk is a <strong className="text-foreground">full</strong> backup; later runs are{" "}
          <strong className="text-foreground">incremental</strong> (only new/changed files). Mount your removable disk and
          point the path at it.
        </p>
      </div>

      <Card><CardContent className="space-y-3 p-4">
        <label className="text-sm font-medium">Backup target (a directory the server can write to)</label>
        <div className="flex gap-2">
          <Input value={path} onChange={(e) => setPath(e.target.value)} placeholder="/backups or /mnt/usb-drive" className="font-mono" />
          <Button variant="outline" onClick={check} disabled={!path || busy !== null}>
            {busy === "status" ? <Loader2 size={16} className="animate-spin" /> : <Database size={16} />} Scan
          </Button>
        </div>

        {status && (
          <div className="rounded-lg border border-border bg-surface-2/50 p-3 text-sm">
            <div className="flex flex-wrap items-center gap-x-6 gap-y-1">
              <span>Existing backup: <strong>{status.exists ? "yes" : "none"}</strong></span>
              <span>Tracked files: <strong>{status.total_files}</strong></span>
              <span>Next run: <strong className="capitalize text-primary">{status.next_mode}</strong></span>
              {status.last_backup_at && <span>Last: <strong>{timeAgo(status.last_backup_at)}</strong></span>}
            </div>
            {(status.archives?.length ?? 0) > 0 && (
              <div className="mt-2 max-h-40 overflow-y-auto border-t border-border/50 pt-2 font-mono text-xs text-muted">
                {(status.archives ?? []).slice().reverse().map((a) => (
                  <div key={a.name} className="flex justify-between gap-4 py-0.5">
                    <span className="truncate">{a.name}</span>
                    <span className="shrink-0">{a.mode} · {a.count} files · {formatBytes(a.bytes)}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        <div className="flex flex-wrap gap-2 pt-1">
          <Button onClick={run} disabled={!path || busy !== null}>
            {busy === "run" ? <Loader2 size={16} className="animate-spin" /> : <HardDriveDownload size={16} />} Run backup
          </Button>
          <Button variant="outline" onClick={restore} disabled={!path || busy !== null}>
            {busy === "restore" ? <Loader2 size={16} className="animate-spin" /> : <HardDriveUpload size={16} />} Restore
          </Button>
        </div>

        {result && result.mode !== "none" && (
          <div className="flex items-center gap-2 rounded-lg border border-success/30 bg-success/10 px-3 py-2 text-sm text-success">
            <RotateCcw size={15} />
            {result.mode === "full" ? "Full" : "Incremental"} backup: {result.files_backed} files ({formatBytes(result.bytes)}) → {result.archive}
          </div>
        )}
      </CardContent></Card>

      <p className="text-center text-xs text-muted">
        Tip: schedule <code className="font-mono">POST /api/v1/admin/backup</code> weekly (cron + an API key) for hands-off backups.
      </p>
    </div>
  );
}
