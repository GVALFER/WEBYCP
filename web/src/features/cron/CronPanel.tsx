import { Button, Input, Label, TextField } from "@heroui/react";
import { Clock3, Plus, Power, Trash2 } from "lucide-react";
import { type FormEvent, useState } from "react";
import useSWR, { useSWRConfig } from "swr";

import {
  createCronJob,
  deleteCronJob,
  fetcher,
  updateCronJob,
  type AccountListResponse,
  type AuthResponse,
  type CronJobListResponse,
} from "../../lib/api";
import { cn } from "../../utils/classnames";
import { errorMessage } from "../../utils/errors";
import { statusClass } from "../../utils/status";

export const CronPanel = ({ session }: { session: AuthResponse }) => {
  const [accountId, setAccountId] = useState("");
  const [name, setName] = useState("");
  const [schedule, setSchedule] = useState("0 * * * *");
  const [command, setCommand] = useState("");
  const [pending, setPending] = useState("");
  const [error, setError] = useState("");
  const { mutate: mutateKey } = useSWRConfig();
  const { data: accounts } = useSWR<AccountListResponse>("accounts", fetcher);
  const { data, mutate } = useSWR<CronJobListResponse>("cron-jobs", fetcher, { refreshInterval: 2_000 });
  const activeAccounts = accounts?.items.filter((account) => account.status === "active" && account.enabled) ?? [];
  const selectedAccount = accountId || activeAccounts[0]?.id || "";

  const run = async (key: string, action: () => Promise<unknown>) => {
    setPending(key);
    setError("");
    try {
      await action();
      await Promise.all([mutate(), mutateKey("jobs")]);
    } catch (requestError) {
      setError(await errorMessage(requestError));
    } finally {
      setPending("");
    }
  };

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    await run("create", async () => {
      await createCronJob({ accountId: selectedAccount, name, schedule, command, enabled: true }, session.csrfToken);
      setName("");
      setCommand("");
    });
  };

  return <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_22rem]">
    <section className="overflow-hidden rounded-2xl border border-divider bg-surface"><div className="border-b border-divider px-6 py-5"><h2 className="text-lg font-semibold">Cron jobs</h2><div className="mt-1 text-sm text-foreground-500">Commands run as the hosting account from its home directory.</div>{error && <div className="mt-4 rounded-xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">{error}</div>}</div><div className="divide-y divide-divider">{data?.items.length ? data.items.map((item) => <div key={item.id} className="flex flex-col gap-4 px-6 py-5 sm:flex-row sm:items-center sm:justify-between"><div className="flex items-center gap-4"><Clock3 className="size-5 text-foreground-400" /><div><div className="font-medium">{item.name}</div><div className="mt-1 font-mono text-xs text-foreground-400">{item.schedule} · {item.command}</div></div></div><div className="flex items-center gap-2"><span className={cn("rounded-full px-2 py-1 text-xs", statusClass(item.status))}>{item.status}</span><Button isIconOnly size="sm" variant="tertiary" aria-label={item.enabled ? `Disable ${item.name}` : `Enable ${item.name}`} isDisabled={pending === item.id || item.status === "pending"} onPress={() => void run(item.id, () => updateCronJob(item.id, { accountId: item.accountId, name: item.name, schedule: item.schedule, command: item.command, enabled: !item.enabled }, session.csrfToken))}><Power className="size-4" /></Button><Button isIconOnly size="sm" variant="danger-soft" aria-label={`Delete ${item.name}`} isDisabled={pending === item.id} onPress={() => { if (window.confirm(`Delete cron job ${item.name}?`)) void run(item.id, () => deleteCronJob(item.id, session.csrfToken)); }}><Trash2 className="size-4" /></Button></div></div>) : <div className="px-6 py-12 text-center text-sm text-foreground-400">No cron jobs yet.</div>}</div></section>
    <form className="h-fit rounded-2xl border border-divider bg-surface p-6" onSubmit={submit}><h2 className="text-lg font-semibold">Add cron job</h2><label className="mt-5 block space-y-2 text-sm"><span>Hosting account</span><select className="h-10 w-full rounded-lg border border-divider bg-background px-3" value={selectedAccount} onChange={(event) => setAccountId(event.currentTarget.value)}>{activeAccounts.map((account) => <option key={account.id} value={account.id}>{account.name}</option>)}</select></label><TextField className="mt-4" fullWidth isRequired><Label>Name</Label><Input value={name} onChange={(event) => setName(event.currentTarget.value)} /></TextField><TextField className="mt-4" fullWidth isRequired><Label>Schedule</Label><Input className="font-mono" value={schedule} onChange={(event) => setSchedule(event.currentTarget.value)} /></TextField><TextField className="mt-4" fullWidth isRequired><Label>Command</Label><Input className="font-mono" value={command} onChange={(event) => setCommand(event.currentTarget.value)} /></TextField><Button className="mt-5" type="submit" variant="primary" fullWidth isDisabled={!selectedAccount || pending !== ""}><Plus className="size-4" />Add cron job</Button></form>
  </div>;
};
