import { Button, Input, Label, TextField } from "@heroui/react";
import { ArchiveRestore, HardDrive, Play, Plus, Trash2 } from "lucide-react";
import { type FormEvent, useState } from "react";
import useSWR, { useSWRConfig } from "swr";

import {
  createBackupPlan,
  deleteBackupArtifact,
  deleteBackupPlan,
  fetcher,
  previewBackup,
  restoreBackup,
  runBackupPlan,
  type AccountListResponse,
  type AuthResponse,
  type BackupArtifactListResponse,
  type BackupPlanListResponse,
  type BackupRunListResponse,
} from "../../lib/api";
import { cn } from "../../utils/classnames";
import { formatDate } from "../../utils/date";
import { errorMessage } from "../../utils/errors";
import { statusClass } from "../../utils/status";

export const BackupsPanel = ({ session }: { session: AuthResponse }) => {
  const [accountId, setAccountId] = useState("");
  const [name, setName] = useState("Daily backup");
  const [schedule, setSchedule] = useState("0 3 * * *");
  const [retention, setRetention] = useState("7");
  const [files, setFiles] = useState(true);
  const [databases, setDatabases] = useState(true);
  const [pending, setPending] = useState("");
  const [error, setError] = useState("");
  const { mutate: mutateKey } = useSWRConfig();
  const { data: accounts } = useSWR<AccountListResponse>("accounts", fetcher);
  const { data: plans, mutate: mutatePlans } = useSWR<BackupPlanListResponse>("backup-plans", fetcher);
  const { data: runs, mutate: mutateRuns } = useSWR<BackupRunListResponse>("backup-runs", fetcher, { refreshInterval: 3_000 });
  const { data: artifacts, mutate: mutateArtifacts } = useSWR<BackupArtifactListResponse>("backup-artifacts", fetcher, { refreshInterval: 5_000 });
  const activeAccounts = accounts?.items.filter((account) => account.status === "active" && account.enabled) ?? [];
  const selectedAccount = accountId || activeAccounts[0]?.id || "";

  const refresh = () => Promise.all([mutatePlans(), mutateRuns(), mutateArtifacts(), mutateKey("jobs")]);
  const run = async (key: string, action: () => Promise<unknown>) => {
    setPending(key);
    setError("");
    try {
      await action();
      await refresh();
    } catch (requestError) {
      setError(await errorMessage(requestError));
    } finally {
      setPending("");
    }
  };

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    await run("create", async () => {
      await createBackupPlan({ accountId: selectedAccount, name, schedule, retentionCount: Number(retention), includeFiles: files, includeDatabases: databases, enabled: true }, session.csrfToken);
    });
  };

  const restore = async (id: string) => {
    await run(id, async () => {
      const manifest = await previewBackup(id);
      if (!window.confirm(`Restore this verified backup with ${manifest.entries.length} entries? Existing files and databases may be overwritten.`)) return;
      await restoreBackup(id, { files: manifest.files, databases: manifest.databases, metadata: manifest.metadata }, session.csrfToken);
    });
  };

  return <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_22rem]">
    <div className="space-y-6">
      {error && <div className="rounded-xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">{error}</div>}
      <section className="overflow-hidden rounded-2xl border border-divider bg-surface"><div className="border-b border-divider px-6 py-5"><h2 className="text-lg font-semibold">Backup plans</h2><div className="mt-1 text-sm text-foreground-500">Scheduled and on-demand local backups with retention.</div></div><div className="divide-y divide-divider">{plans?.items.length ? plans.items.map((plan) => <div key={plan.id} className="flex flex-col gap-4 px-6 py-5 sm:flex-row sm:items-center sm:justify-between"><div className="flex items-center gap-4"><HardDrive className="size-5 text-foreground-400" /><div><div className="font-medium">{plan.name}</div><div className="mt-1 text-xs text-foreground-400">{plan.schedule || "Manual only"} · keep {plan.retentionCount} · next {formatDate(plan.nextRunAt)}</div></div></div><div className="flex items-center gap-2"><Button size="sm" variant="secondary" isDisabled={pending !== ""} onPress={() => void run(plan.id, () => runBackupPlan(plan.id, session.csrfToken))}><Play className="size-4" />Run now</Button><Button isIconOnly size="sm" variant="danger-soft" aria-label={`Delete ${plan.name}`} isDisabled={pending !== ""} onPress={() => { if (window.confirm(`Delete plan ${plan.name}? Existing artifacts are retained.`)) void run(plan.id, () => deleteBackupPlan(plan.id, session.csrfToken)); }}><Trash2 className="size-4" /></Button></div></div>) : <div className="px-6 py-12 text-center text-sm text-foreground-400">No backup plans yet.</div>}</div></section>
      <section className="overflow-hidden rounded-2xl border border-divider bg-surface"><div className="border-b border-divider px-6 py-5"><h2 className="text-lg font-semibold">Verified artifacts</h2></div><div className="divide-y divide-divider">{artifacts?.items.length ? artifacts.items.map((artifact) => <div key={artifact.id} className="flex items-center justify-between gap-4 px-6 py-4"><div><div className="font-medium">{formatDate(artifact.createdAt)}</div><div className="mt-1 text-xs text-foreground-400">{(artifact.size / 1_048_576).toFixed(2)} MB · SHA-256 {artifact.checksum.slice(0, 12)}…</div></div><div className="flex items-center gap-2"><Button size="sm" variant="secondary" isDisabled={pending !== ""} onPress={() => void restore(artifact.id)}><ArchiveRestore className="size-4" />Preview & restore</Button><Button isIconOnly size="sm" variant="danger-soft" aria-label="Delete backup artifact" isDisabled={pending !== ""} onPress={() => { if (window.confirm("Permanently delete this local backup artifact?")) void run(artifact.id, () => deleteBackupArtifact(artifact.id, session.csrfToken)); }}><Trash2 className="size-4" /></Button></div></div>) : <div className="px-6 py-10 text-center text-sm text-foreground-400">No completed artifacts yet.</div>}</div></section>
      <section className="rounded-2xl border border-divider bg-surface p-6"><h2 className="text-lg font-semibold">Recent runs</h2><div className="mt-4 space-y-3">{runs?.items.slice(0, 10).map((item) => <div key={item.id} className="flex items-center justify-between rounded-xl bg-default/5 px-4 py-3 text-sm"><div>{formatDate(item.createdAt)}</div><span className={cn("rounded-full px-2 py-1 text-xs", statusClass(item.status))}>{item.status}</span></div>)}</div></section>
    </div>
    <form className="h-fit rounded-2xl border border-divider bg-surface p-6" onSubmit={submit}><h2 className="text-lg font-semibold">Create backup plan</h2><label className="mt-5 block space-y-2 text-sm"><span>Hosting account</span><select className="h-10 w-full rounded-lg border border-divider bg-background px-3" value={selectedAccount} onChange={(event) => setAccountId(event.currentTarget.value)}>{activeAccounts.map((account) => <option key={account.id} value={account.id}>{account.name}</option>)}</select></label><TextField className="mt-4" fullWidth isRequired><Label>Name</Label><Input value={name} onChange={(event) => setName(event.currentTarget.value)} /></TextField><TextField className="mt-4" fullWidth><Label>Schedule (blank for manual)</Label><Input className="font-mono" value={schedule} onChange={(event) => setSchedule(event.currentTarget.value)} /></TextField><TextField className="mt-4" fullWidth isRequired><Label>Retention</Label><Input type="number" min="1" max="100" value={retention} onChange={(event) => setRetention(event.currentTarget.value)} /></TextField><label className="mt-5 flex items-center gap-3 text-sm"><input type="checkbox" checked={files} onChange={(event) => setFiles(event.currentTarget.checked)} />Files</label><label className="mt-3 flex items-center gap-3 text-sm"><input type="checkbox" checked={databases} onChange={(event) => setDatabases(event.currentTarget.checked)} />Databases</label><Button className="mt-6" type="submit" variant="primary" fullWidth isDisabled={!selectedAccount || (!files && !databases) || pending !== ""}><Plus className="size-4" />Create plan</Button></form>
  </div>;
};
