"use client";

import { useQuery } from "@tanstack/react-query";
import { filesApi } from "@/lib/endpoints";
import { FileList, PageHeader } from "@/components/files/file-list";

export default function TrashPage() {
  const q = useQuery({ queryKey: ["trash"], queryFn: () => filesApi.trash() });
  return (
    <div className="mx-auto max-w-5xl space-y-4">
      <PageHeader title="Trash" subtitle="Deleted files are purged automatically after the retention window" />
      <FileList files={q.data} loading={q.isLoading} queryKey="trash" emptyLabel="Trash is empty." mode="trash" />
    </div>
  );
}
