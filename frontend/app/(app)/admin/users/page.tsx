"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Ban, CheckCircle2, KeyRound, Plus, ShieldAlert, ShieldCheck, Trash2, UserPlus, Users } from "lucide-react";
import { usersApi, rolesApi } from "@/lib/endpoints";
import { ApiError } from "@/lib/api";
import { PageHeader } from "@/components/files/file-list";
import { Avatar } from "@/components/ui/avatar";
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
  const [form, setForm] = useState({ email: "", username: "", full_name: "", password: "", role: "employee", quota_gb: 15 });
  const isSuper = form.role === "super_admin";
  const [deleting, setDeleting] = useState<import("@/lib/endpoints").AdminUser | null>(null);
  const [transferTo, setTransferTo] = useState("");

  async function confirmDelete() {
    if (!deleting) return;
    if (!transferTo) return toast.error("Select a user to receive the files");
    try {
      await usersApi.remove(deleting.id, transferTo);
      toast.success("User removed; files transferred");
      setDeleting(null); setTransferTo(""); refresh();
    } catch (e) { toast.error(e instanceof ApiError ? e.message : "Could not delete user"); }
  }
  async function enableUser(id: string) {
    try { await usersApi.enable(id); toast.success("Account re-enabled"); refresh(); }
    catch (e) { toast.error(e instanceof ApiError ? e.message : "Only a super admin can re-enable accounts"); }
  }

  const refresh = () => qc.invalidateQueries({ queryKey: ["users"] });

  async function create() {
    // Client-side validation mirrors the backend so users get a clear reason.
    if (!form.full_name.trim()) return toast.error("Full name is required");
    if (!/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(form.email)) return toast.error("Enter a valid email address");
    if (form.username.trim().length < 3) return toast.error("Username must be at least 3 characters");
    if (form.password.length < 12) return toast.error("Password must be at least 12 characters");
    if (!isSuper && (!form.quota_gb || form.quota_gb <= 0)) return toast.error("Set a storage quota (GB)");

    // Super admins are unlimited (quota 0); everyone else gets a fixed quota.
    const storage_quota = isSuper ? 0 : Math.round(form.quota_gb * 1024 ** 3);
    try {
      await usersApi.create({ email: form.email, username: form.username, password: form.password, full_name: form.full_name, role: form.role, storage_quota });
      toast.success("User created");
      setOpen(false);
      setForm({ email: "", username: "", full_name: "", password: "", role: "employee", quota_gb: 15 });
      refresh();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : "Could not create user");
    }
  }

  async function resetPassword(id: string, username: string) {
    const pw = prompt(`New password for ${username} (min 12 chars)`);
    if (!pw) return;
    if (pw.length < 12) { toast.error("Password must be at least 12 characters"); return; }
    try { await usersApi.resetPassword(id, pw); toast.success("Password reset — user's sessions revoked"); }
    catch { toast.error("Could not reset password"); }
  }

  return (
    <div className="mx-auto max-w-5xl space-y-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <PageHeader icon={Users} title="Users" subtitle="Manage accounts, roles, storage quotas and access across your organisation" />
        <Button size="sm" onClick={() => setOpen((o) => !o)}><UserPlus size={16} /> Add user</Button>
      </div>

      {open && (
        <Card><CardContent className="p-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <Input placeholder="Full name" value={form.full_name} onChange={(e) => setForm({ ...form, full_name: e.target.value })} />
            <Input placeholder="Email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} />
            <Input placeholder="Username" value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} />
            <Input placeholder="Temp password (min 12)" type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} />
            <select className="h-10 rounded-lg border border-border bg-surface px-3 text-sm" value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })}>
              {roles.data?.map((r) => <option key={r.slug} value={r.slug}>{r.name}</option>)}
            </select>
            <div className="relative">
              <Input
                type="number" min={1} placeholder="Max storage (GB)"
                value={isSuper ? "" : form.quota_gb}
                disabled={isSuper}
                onChange={(e) => setForm({ ...form, quota_gb: Number(e.target.value) })}
              />
              <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted">
                {isSuper ? "Unlimited" : "GB"}
              </span>
            </div>
            <Button size="sm" onClick={create}><Plus size={16} /> Create</Button>
          </div>

          {/* RBAC access preview for the selected role */}
          <div className="mt-4 rounded-lg border border-border bg-surface-2 p-3">
            {(() => {
              const role = roles.data?.find((r) => r.slug === form.role);
              return (
                <>
                  <div className="mb-2 flex items-center gap-2">
                    <ShieldCheck size={14} className="text-primary" />
                    <p className="text-sm font-medium">Access granted by <span className="capitalize">{role?.name ?? form.role}</span></p>
                  </div>
                  {role?.description && <p className="mb-2 text-xs text-muted">{role.description}</p>}
                  {role && role.permissions.length > 0 ? (
                    <div className="flex flex-wrap gap-1.5">
                      {role.permissions.map((p) => <Badge key={p}>{p}</Badge>)}
                    </div>
                  ) : (
                    <p className="text-xs text-muted">This role grants no explicit permissions.</p>
                  )}
                  <p className="mt-2 font-mono text-[11px] text-muted">
                    {role?.permissions.length ?? 0} permission{(role?.permissions.length ?? 0) === 1 ? "" : "s"}
                  </p>
                </>
              );
            })()}
          </div>
        </CardContent></Card>
      )}

      {users.isLoading && <Skeleton className="h-48 w-full" />}
      <div className="space-y-2">
        {users.data?.map((u) => (
          <Card key={u.id}>
            <CardContent className="flex flex-col gap-3 p-4 sm:flex-row sm:items-center">
              <div className="flex min-w-0 flex-1 items-center gap-3">
                <Avatar userId={u.id} name={u.full_name || u.username} hasAvatar={u.has_avatar} size={40} />
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <p className="truncate font-medium">{u.full_name || u.username}</p>
                    {!u.is_active && <span className="rounded-full bg-danger/10 px-2 py-0.5 text-[10px] font-medium text-danger">Disabled</span>}
                    {u.is_locked && <span className="rounded-full bg-warning/10 px-2 py-0.5 text-[10px] font-medium text-warning">Locked</span>}
                  </div>
                  <div className="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted">
                    <span className="truncate">{u.email}</span>
                    <Badge className="capitalize">{u.role.replace("_", " ")}</Badge>
                    <span>· {formatBytes(u.storage_used)} used</span>
                    {u.last_login_at && <span>· seen {timeAgo(u.last_login_at)}</span>}
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-2 self-end sm:self-auto">
                <select
                  className="h-8 rounded-md border border-border bg-surface px-2 text-xs"
                  value={u.role}
                  onChange={async (e) => { await usersApi.setRole(u.id, e.target.value).then(() => { toast.success("Role updated"); refresh(); }); }}
                >
                  {roles.data?.map((r) => <option key={r.slug} value={r.slug}>{r.name}</option>)}
                </select>
                <button title="Reset password"
                  onClick={() => resetPassword(u.id, u.username)}
                  className="flex h-8 w-8 items-center justify-center rounded-md text-muted hover:bg-surface-2 hover:text-foreground">
                  <KeyRound size={16} />
                </button>
                <button title={u.is_active ? "Disable" : "Re-enable (super admin)"}
                  onClick={() => u.is_active
                    ? usersApi.setActive(u.id, false).then(() => { toast.success("Disabled"); refresh(); })
                    : enableUser(u.id)}
                  className="flex h-8 w-8 items-center justify-center rounded-md hover:bg-surface-2">
                  {u.is_active ? <Ban size={16} className="text-warning" /> : <CheckCircle2 size={16} className="text-success" />}
                </button>
                <button title="Delete (transfers files)" onClick={() => { setDeleting(u); setTransferTo(""); }}
                  className="flex h-8 w-8 items-center justify-center rounded-md text-danger hover:bg-surface-2"><Trash2 size={16} /></button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Delete-with-transfer modal */}
      {deleting && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={() => setDeleting(null)}>
          <div className="w-full max-w-md rounded-2xl border border-border bg-surface p-5 shadow-xl" onClick={(e) => e.stopPropagation()}>
            <div className="mb-3 flex items-center gap-2 text-danger">
              <ShieldAlert size={18} />
              <h3 className="text-base font-semibold">Delete {deleting.full_name || deleting.username}</h3>
            </div>
            <p className="text-sm text-muted">
              All of this user&apos;s files and folders must be transferred to another user. The recipient
              has <span className="font-medium text-foreground">30 days</span> to keep or delete them —
              nothing is auto-deleted, and their account is disabled if they take no action.
            </p>
            <label className="mt-4 block text-sm font-medium">Transfer files to</label>
            <select
              className="mt-1.5 h-10 w-full rounded-lg border border-border bg-surface px-3 text-sm"
              value={transferTo}
              onChange={(e) => setTransferTo(e.target.value)}
            >
              <option value="">Select a user…</option>
              {users.data?.filter((x) => x.id !== deleting.id && x.is_active).map((x) => (
                <option key={x.id} value={x.id}>{x.full_name || x.username} ({x.email})</option>
              ))}
            </select>
            <div className="mt-5 flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setDeleting(null)}>Cancel</Button>
              <Button variant="danger" size="sm" onClick={confirmDelete}>Delete &amp; transfer</Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
