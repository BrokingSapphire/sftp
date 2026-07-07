"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { motion } from "motion/react";
import { UsersRound, Plus, UserPlus, Trash2, X, Crown, Shield, Eye, User as UserIcon, Loader2 } from "lucide-react";
import { teamsApi, type Team, type TeamMember } from "@/lib/endpoints";
import { ApiError } from "@/lib/api";
import { PageHeader } from "@/components/files/file-list";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Avatar } from "@/components/ui/avatar";
import { Skeleton } from "@/components/ui/misc";
import { EmptyState } from "@/components/ui/empty-state";
import { StaggerList, StaggerItem } from "@/components/motion";
import { formatBytes } from "@/lib/utils";

const roleIcon: Record<string, React.ElementType> = { owner: Crown, admin: Shield, member: UserIcon, viewer: Eye };

// Preset team colours (first is the default).
const TEAM_COLORS = ["#064D51", "#2563eb", "#7c3aed", "#db2777", "#ea580c", "#16a34a", "#0891b2", "#ca8a04"];

export default function TeamsPage() {
  const qc = useQueryClient();
  const q = useQuery({ queryKey: ["teams"], queryFn: () => teamsApi.list() });
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState({ name: "", description: "", quota_gb: 0, color: TEAM_COLORS[0] });
  const [managing, setManaging] = useState<Team | null>(null);

  const teams = q.data ?? [];
  const refresh = () => qc.invalidateQueries({ queryKey: ["teams"] });

  async function create() {
    if (!form.name.trim()) return toast.error("Team name is required");
    try {
      await teamsApi.create(form.name, form.description, form.quota_gb > 0 ? Math.round(form.quota_gb * 1024 ** 3) : 0, form.color);
      toast.success("Team created");
      setCreating(false); setForm({ name: "", description: "", quota_gb: 0, color: TEAM_COLORS[0] }); refresh();
    } catch (e) { toast.error(e instanceof ApiError ? e.message : "Could not create team"); }
  }
  async function remove(t: Team) {
    if (!confirm(`Delete team “${t.name}” and its shared drive? This cannot be undone.`)) return;
    try { await teamsApi.remove(t.id); toast.success("Team deleted"); refresh(); }
    catch (e) { toast.error(e instanceof ApiError ? e.message : "Could not delete"); }
  }

  return (
    <div className="mx-auto max-w-5xl space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <PageHeader title="Team Spaces" subtitle="Shared drives owned by a group, with roles and delegated admin" />
        <Button size="sm" onClick={() => setCreating((v) => !v)}><Plus size={16} /> New team</Button>
      </div>

      {creating && (
        <Card><CardContent className="grid grid-cols-1 gap-3 p-4 sm:grid-cols-[1fr_1fr_10rem_auto]">
          <Input placeholder="Team name" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          <Input placeholder="Description (optional)" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
          <Input type="number" min={0} placeholder="Quota (GB, 0=∞)" value={form.quota_gb || ""} onChange={(e) => setForm({ ...form, quota_gb: Number(e.target.value) })} />
          <Button size="sm" onClick={create}><Plus size={16} /> Create</Button>
          <div className="col-span-full flex items-center gap-2">
            <span className="text-xs text-muted">Colour:</span>
            {TEAM_COLORS.map((c) => (
              <button key={c} type="button" onClick={() => setForm({ ...form, color: c })} aria-label={`Colour ${c}`}
                className={`h-6 w-6 rounded-full transition-transform hover:scale-110 ${form.color === c ? "ring-2 ring-offset-2 ring-offset-surface" : ""}`}
                style={{ backgroundColor: c, ...(form.color === c ? { boxShadow: `0 0 0 2px ${c}` } : {}) }} />
            ))}
          </div>
        </CardContent></Card>
      )}

      {q.isLoading && <Skeleton className="h-40 w-full" />}

      {!q.isLoading && teams.length === 0 && (
        <EmptyState
          icon={UsersRound}
          title="No teams yet — assemble the crew"
          subtitle="Create a Team Space so a department can share one drive. Great files are better together (and so is the blame)."
          action={<Button size="sm" onClick={() => setCreating(true)}><Plus size={16} /> Create your first team</Button>}
        />
      )}

      {teams.length > 0 && (
        <StaggerList>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            {teams.map((t) => {
              const RI = roleIcon[t.member_role ?? "member"] ?? UserIcon;
              return (
                <StaggerItem key={t.id}>
                  <Card className="group h-full">
                    <CardContent className="flex h-full flex-col gap-3 p-4">
                      <div className="flex items-start gap-3">
                        <span className="flex h-11 w-11 items-center justify-center rounded-xl text-white"
                          style={{ backgroundColor: t.color || "#064D51" }}><UsersRound size={20} /></span>
                        <div className="min-w-0 flex-1">
                          <p className="truncate font-semibold">{t.name}</p>
                          {t.description && <p className="truncate text-xs text-muted">{t.description}</p>}
                        </div>
                        <span className="flex items-center gap-1 rounded-full bg-surface-2 px-2 py-0.5 text-[11px] font-medium capitalize text-muted"><RI size={11} /> {t.member_role}</span>
                      </div>
                      <div className="mt-auto flex items-center justify-between text-xs text-muted">
                        <span>{t.member_count ?? 1} member{(t.member_count ?? 1) === 1 ? "" : "s"} · {t.storage_quota > 0 ? formatBytes(t.storage_quota) : "unlimited"}</span>
                        <div className="flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                          <Button variant="outline" size="sm" onClick={() => setManaging(t)}><UserPlus size={13} /> Members</Button>
                          {t.member_role === "owner" && (
                            <button title="Delete team" onClick={() => remove(t)} className="flex h-7 w-7 items-center justify-center rounded-md text-danger hover:bg-surface-2"><Trash2 size={14} /></button>
                          )}
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                </StaggerItem>
              );
            })}
          </div>
        </StaggerList>
      )}

      {managing && <MembersModal team={managing} onClose={() => setManaging(null)} onChanged={refresh} />}
    </div>
  );
}

function MembersModal({ team, onClose, onChanged }: { team: Team; onClose: () => void; onChanged: () => void }) {
  const qc = useQueryClient();
  const q = useQuery({ queryKey: ["team-members", team.id], queryFn: () => teamsApi.members(team.id) });
  const [email, setEmail] = useState("");
  const [role, setRole] = useState("member");
  const [busy, setBusy] = useState(false);
  const canManage = team.member_role === "owner" || team.member_role === "admin";
  const refresh = () => { qc.invalidateQueries({ queryKey: ["team-members", team.id] }); onChanged(); };

  async function add() {
    if (!email) return;
    setBusy(true);
    try { await teamsApi.addMember(team.id, email, role); toast.success("Member added"); setEmail(""); refresh(); }
    catch (e) { toast.error(e instanceof ApiError ? (e.message.includes("not found") ? "No user with that email" : e.message) : "Could not add"); }
    finally { setBusy(false); }
  }
  async function kick(m: TeamMember) {
    try { await teamsApi.removeMember(team.id, m.user_id); toast.success("Removed"); refresh(); }
    catch (e) { toast.error(e instanceof ApiError ? e.message : "Could not remove"); }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onClose}>
      <motion.div initial={{ opacity: 0, scale: 0.97 }} animate={{ opacity: 1, scale: 1 }}
        className="max-h-[85vh] w-full max-w-md overflow-hidden rounded-2xl border border-border bg-surface shadow-xl" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between border-b border-border px-5 py-3">
          <div><h3 className="text-sm font-semibold">Members</h3><p className="truncate text-xs text-muted">{team.name}</p></div>
          <button onClick={onClose} className="text-muted hover:text-foreground"><X size={18} /></button>
        </div>

        {canManage && (
          <div className="flex items-center gap-2 border-b border-border px-4 py-3">
            <Input type="email" placeholder="Add by email" value={email} onChange={(e) => setEmail(e.target.value)} onKeyDown={(e) => e.key === "Enter" && add()} />
            <select value={role} onChange={(e) => setRole(e.target.value)} className="h-10 rounded-lg border border-border bg-surface px-2 text-sm">
              <option value="admin">Admin</option><option value="member">Member</option><option value="viewer">Viewer</option>
            </select>
            <Button size="sm" className="h-10" onClick={add} disabled={busy || !email}>{busy ? <Loader2 size={15} className="animate-spin" /> : <UserPlus size={15} />}</Button>
          </div>
        )}

        <div className="max-h-[50vh] overflow-y-auto p-2">
          {q.isLoading && <Skeleton className="m-2 h-24 w-full" />}
          {(q.data ?? []).map((m) => {
            const RI = roleIcon[m.role] ?? UserIcon;
            return (
              <div key={m.user_id} className="flex items-center gap-2.5 rounded-lg px-2 py-2 hover:bg-surface-2">
                <Avatar userId={m.user_id} name={m.name} hasAvatar={m.has_avatar} size={32} />
                <div className="min-w-0 flex-1"><p className="truncate text-sm font-medium">{m.name}</p><p className="truncate text-xs text-muted">{m.email}</p></div>
                <span className="flex items-center gap-1 text-xs capitalize text-muted"><RI size={12} /> {m.role}</span>
                {canManage && m.role !== "owner" && (
                  <button title="Remove" onClick={() => kick(m)} className="flex h-7 w-7 items-center justify-center rounded-md text-danger hover:bg-border"><Trash2 size={14} /></button>
                )}
              </div>
            );
          })}
        </div>
      </motion.div>
    </div>
  );
}
