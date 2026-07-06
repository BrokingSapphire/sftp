"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Trash2 } from "lucide-react";
import { filesApi } from "@/lib/endpoints";
import { FileList, PageHeader } from "@/components/files/file-list";
import { Button } from "@/components/ui/button";

export default function TrashPage() {
  const qc = useQueryClient();
  const q = useQuery({ queryKey: ["trash"], queryFn: () => filesApi.trash() });
  const count = q.data?.length ?? 0;

  async function emptyTrash() {
    if (!confirm("Permanently delete everything in Trash? This cannot be undone.")) return;
    try {
      await filesApi.emptyTrash();
      toast.success("Trash emptied");
      qc.invalidateQueries({ queryKey: ["trash"] });
    } catch { toast.error("Could not empty trash"); }
  }

  return (
    <div className="mx-auto max-w-5xl space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <PageHeader title="Trash" subtitle="Deleted files are purged automatically after the retention window" />
        {count > 0 && (
          <Button variant="danger" size="sm" onClick={emptyTrash}>
            <Trash2 size={16} /> Clear all
          </Button>
        )}
      </div>
      <FileList files={q.data} loading={q.isLoading} queryKey="trash" emptyLabel="Squeaky clean" emptyIcon={Trash2} emptySubtitle="Your trash is emptier than a Monday-morning inbox. Nothing to see here." mode="trash" />
    </div>
  );
}
