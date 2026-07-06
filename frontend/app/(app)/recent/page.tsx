"use client";

import { useQuery } from "@tanstack/react-query";
import { filesApi } from "@/lib/endpoints";
import { FileList, PageHeader } from "@/components/files/file-list";
import { Clock } from "lucide-react";

export default function RecentPage() {
  const q = useQuery({ queryKey: ["recent"], queryFn: () => filesApi.recent() });
  return (
    <div className="mx-auto max-w-5xl space-y-4">
      <PageHeader title="Recent" subtitle="Your most recently added files" />
      <FileList files={q.data} loading={q.isLoading} queryKey="recent" emptyLabel="Nothing here… yet" emptyIcon={Clock} emptySubtitle="Upload something and it will show up here faster than you can say 'where did I put that file'." />
    </div>
  );
}
