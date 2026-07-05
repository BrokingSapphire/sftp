"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Ban, CheckCircle2, Plus, Trash2, UserPlus } from "lucide-react";
import { usersApi, rolesApi } from "@/lib/endpoints";
import { PageHeader } from "@/components/files/file-list";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge, Skeleton } from "@/components/ui/misc";
import { formatBytes, timeAgo } from "@/lib/utils";

export default function AdminUsersPage() {
  const qc = useQueryClient();
  const users = useQuery({ queryKey: ["users"], queryFn: () => usersApi.list() });
  const roles = useQuery({ queryKey: ["roles"], queryFn: () => rolesApi.list() });
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ email: "", username: "", full_name: "", password: "", role: "employee" });

  const refresh = () => qc.invalidateQueries({ queryKey: ["users"] });

  async function create() {
    try {
      await usersApi.create(form);
      toast.success("User created");
      setOpen(false);
      setForm({ email: "", username: "", full_name: "", password: "", role: "employee" });
      refresh();
    } catch { toast.error("Could not create user"); }
  }

  return (
    <div className="mx-auto max-w-5xl space-y-4">
      <div className="flex items-center justify-between">
        <PageHeader title="Users" subtitle="Manage accounts, roles and access" />
        <Button size="sm" onClick={() => setOpen((o) => !o)}><UserPlus size={16} /> Add user</Button>
      </div>

      {open && (
        <Card><CardContent className="grid grid-cols-1 gap-3 p-4 sm:grid-cols-2">
          <Input placeholder="Full name" value={form.full_name} onChange={(e) => setForm({ ...form, full_name: e.target.value })} />
          <Input placeholder="Email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} />
          <Input placeholder="Username" value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} />
          <Input placeholder="Temp password (min 12)" type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} />
          <select className="h-10 rounded-lg border border-border bg-surface px-3 text-sm" value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })}>
            {roles.data?.map((r) => <option key={r.slug} value={r.slug}>{r.name}</option>)}
          </select>
          <Button size="sm" onClick={create}><Plus size={16} /> Create</Button>
        </CardContent></Card>
      )}

      {users.isLoading && <Skeleton className="h-48 w-full" />}
      <div className="space-y-2">
        {users.data?.map((u) => (
          <Card key={u.id}>
            <CardContent className="flex items-center gap-4 p-4">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary/15 text-sm font-semibold text-primary">
                {(u.full_name || u.username).slice(0, 2).toUpperCase()}
              </div>
              <div className="min-w-0 flex-1">
                <p className="truncate font-medium">{u.full_name || u.username}</p>
                <div className="mt-0.5 flex flex-wrap items-center gap-2 text-xs text-muted">
                  <span>{u.email}</span>
                  <Badge className="capitalize">{u.role.replace("_", " ")}</Badge>
                  {!u.is_active && <span className="text-danger">disabled</span>}
                  {u.is_locked && <span className="text-warning">locked</span>}
                  <span>· {formatBytes(u.storage_used)} used</span>
                  {u.last_login_at && <span>· seen {timeAgo(u.last_login_at)}</span>}
                </div>
              </div>
              <select
                className="h-8 rounded-md border border-border bg-surface px-2 text-xs"
                value={u.role}
                onChange={async (e) => { await usersApi.setRole(u.id, e.target.value).then(() => { toast.success("Role updated"); refresh(); }); }}
              >
                {roles.data?.map((r) => <option key={r.slug} value={r.slug}>{r.name}</option>)}
              </select>
              <button title={u.is_active ? "Disable" : "Enable"}
                onClick={() => usersApi.setActive(u.id, !u.is_active).then(() => { toast.success("Updated"); refresh(); })}
                className="flex h-8 w-8 items-center justify-center rounded-md hover:bg-surface-2">
                {u.is_active ? <Ban size={16} className="text-warning" /> : <CheckCircle2 size={16} className="text-success" />}
              </button>
              <button title="Delete" onClick={() => confirm(`Delete ${u.username}?`) && usersApi.remove(u.id).then(() => { toast.success("Deleted"); refresh(); })}
                className="flex h-8 w-8 items-center justify-center rounded-md text-danger hover:bg-surface-2"><Trash2 size={16} /></button>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
