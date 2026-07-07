"use client";

import { useQuery } from "@tanstack/react-query";
import { filesApi } from "@/lib/endpoints";
import { FileList, PageHeader } from "@/components/files/file-list";
import { Star } from "lucide-react";

export default function StarredPage() {
  const q = useQuery({ queryKey: ["starred"], queryFn: () => filesApi.starred() });
  return (
    <div className="mx-auto max-w-5xl space-y-4">
      <PageHeader icon={Star} title="Starred" subtitle="Files you have starred for quick access" />
      <FileList files={q.data} loading={q.isLoading} queryKey="starred" emptyLabel="Nothing starred yet" emptyIcon={Star} emptySubtitle="Star the files you love so they're one click away. Playing favourites is encouraged here." />
    </div>
  );
}
