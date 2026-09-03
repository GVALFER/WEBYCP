import { Button, Input, Label, TextField } from "@heroui/react";
import { Database as DatabaseIcon, KeyRound, Plus, Trash2, UserRound } from "lucide-react";
import { type FormEvent, useState } from "react";
import useSWR, { useSWRConfig } from "swr";

import {
  createDatabase,
  createDatabaseUser,
  deleteDatabase,
  deleteDatabaseUser,
  fetcher,
  setDatabaseGrant,
  type AccountListResponse,
  type AuthResponse,
  type DatabaseGrantListResponse,
  type DatabaseListResponse,
  type DatabaseUserListResponse,
} from "../../lib/api";
import { errorMessage } from "../../utils/errors";
import { cn } from "../../utils/classnames";
import { statusClass } from "../../utils/status";

export const DatabasesPanel = ({ session }: { session: AuthResponse }) => {
  const [accountId, setAccountId] = useState("");
  const [name, setName] = useState("");
  const [userName, setUserName] = useState("");
  const [databaseId, setDatabaseId] = useState("");
  const [userId, setUserId] = useState("");
  const [password, setPassword] = useState("");
  const [pending, setPending] = useState("");
  const [error, setError] = useState("");
  const { mutate: mutateKey } = useSWRConfig();
  const { data: accounts } = useSWR<AccountListResponse>("accounts", fetcher);
  const { data: databases, mutate: mutateDatabases } = useSWR<DatabaseListResponse>("databases", fetcher, { refreshInterval: 2_000 });
  const { data: users, mutate: mutateUsers } = useSWR<DatabaseUserListResponse>("database-users", fetcher, { refreshInterval: 2_000 });
  const { data: grants, mutate: mutateGrants } = useSWR<DatabaseGrantListResponse>("database-grants", fetcher, { refreshInterval: 2_000 });
  const activeAccounts = accounts?.items.filter((account) => account.status === "active" && account.enabled) ?? [];
  const selectedAccount = accountId || activeAccounts[0]?.id || "";
  const accountDatabases = databases?.items.filter((item) => item.accountId === selectedAccount && item.status === "active") ?? [];
  const accountUsers = users?.items.filter((item) => item.accountId === selectedAccount && item.status === "active") ?? [];
  const selectedDatabase = databaseId || accountDatabases[0]?.id || "";
  const selectedUser = userId || accountUsers[0]?.id || "";

  const run = async (key: string, action: () => Promise<unknown>) => {
    setPending(key);
    setError("");
    try {
      await action();
      await Promise.all([mutateDatabases(), mutateUsers(), mutateGrants(), mutateKey("jobs")]);
    } catch (requestError) {
      setError(await errorMessage(requestError));
    } finally {
      setPending("");
    }
  };

  const submitDatabase = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    await run("database", async () => {
      await createDatabase(selectedAccount, name, session.csrfToken);
      setName("");
    });
  };

  const submitUser = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    await run("user", async () => {
      const result = await createDatabaseUser(selectedAccount, userName, session.csrfToken);
      setPassword(result.password ?? "");
      setUserName("");
    });
  };

  const submitGrant = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!selectedDatabase || !selectedUser) return;
    await run("grant", () => setDatabaseGrant(selectedDatabase, selectedUser, true, session.csrfToken));
  };

  return (
    <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_22rem]">
      <div className="space-y-6">
        {password && (
          <div className="rounded-2xl border border-warning/40 bg-warning/10 p-5">
            <div className="font-medium">Save this password now</div>
            <div className="mt-2 break-all font-mono text-sm">{password}</div>
            <div className="mt-2 text-xs text-foreground-500">It will not be shown again.</div>
          </div>
        )}
        {error && <div className="rounded-xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">{error}</div>}
        <ResourceList
          title="MySQL databases"
          empty="No databases yet."
          items={databases?.items ?? []}
          icon={DatabaseIcon}
          onDelete={(id, itemName) => {
            if (window.confirm(`Delete database ${itemName}? Its data will be removed.`)) void run(id, () => deleteDatabase(id, session.csrfToken));
          }}
          pending={pending}
        />
        <ResourceList
          title="MySQL users"
          empty="No database users yet."
          items={users?.items ?? []}
          icon={UserRound}
          onDelete={(id, itemName) => {
            if (window.confirm(`Delete database user ${itemName}?`)) void run(id, () => deleteDatabaseUser(id, session.csrfToken));
          }}
          pending={pending}
        />
        <section className="rounded-2xl border border-divider bg-surface p-6">
          <h2 className="text-lg font-semibold">Grants</h2>
          <div className="mt-4 space-y-3">
            {grants?.items.length ? grants.items.map((grant) => {
              const database = databases?.items.find((item) => item.id === grant.databaseId);
              const user = users?.items.find((item) => item.id === grant.databaseUserId);
              return <div key={`${grant.databaseId}:${grant.databaseUserId}`} className="flex items-center justify-between rounded-xl bg-default/5 px-4 py-3 text-sm">
                <div><span className="font-medium">{user?.name}</span> → {database?.name}</div>
                <Button isIconOnly size="sm" variant="danger-soft" aria-label="Revoke grant" onPress={() => void run(`grant:${grant.databaseId}`, () => setDatabaseGrant(grant.databaseId, grant.databaseUserId, false, session.csrfToken))}><Trash2 className="size-4" /></Button>
              </div>;
            }) : <div className="text-sm text-foreground-400">No grants yet.</div>}
          </div>
        </section>
      </div>
      <aside className="space-y-6">
        <section className="rounded-2xl border border-divider bg-surface p-6">
          <h2 className="text-lg font-semibold">Create resources</h2>
          <label className="mt-5 block space-y-2 text-sm"><span className="font-medium">Hosting account</span><select className="h-10 w-full rounded-lg border border-divider bg-background px-3" value={selectedAccount} onChange={(event) => setAccountId(event.currentTarget.value)}>{activeAccounts.map((account) => <option key={account.id} value={account.id}>{account.name}</option>)}</select></label>
          <form className="mt-5 space-y-4" onSubmit={submitDatabase}><TextField fullWidth isRequired><Label>Database name</Label><Input value={name} onChange={(event) => setName(event.currentTarget.value)} /></TextField><Button type="submit" variant="primary" fullWidth isDisabled={!selectedAccount || pending !== ""}><Plus className="size-4" />Create database</Button></form>
          <form className="mt-6 space-y-4 border-t border-divider pt-6" onSubmit={submitUser}><TextField fullWidth isRequired><Label>User name</Label><Input value={userName} onChange={(event) => setUserName(event.currentTarget.value)} /></TextField><Button type="submit" variant="secondary" fullWidth isDisabled={!selectedAccount || pending !== ""}><KeyRound className="size-4" />Create user</Button></form>
        </section>
        <form className="rounded-2xl border border-divider bg-surface p-6" onSubmit={submitGrant}>
          <h2 className="text-lg font-semibold">Add grant</h2>
          <label className="mt-5 block space-y-2 text-sm"><span>Database</span><select className="h-10 w-full rounded-lg border border-divider bg-background px-3" value={selectedDatabase} onChange={(event) => setDatabaseId(event.currentTarget.value)}>{accountDatabases.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
          <label className="mt-4 block space-y-2 text-sm"><span>User</span><select className="h-10 w-full rounded-lg border border-divider bg-background px-3" value={selectedUser} onChange={(event) => setUserId(event.currentTarget.value)}>{accountUsers.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
          <Button className="mt-5" type="submit" variant="primary" fullWidth isDisabled={!selectedDatabase || !selectedUser || pending !== ""}>Grant access</Button>
        </form>
      </aside>
    </div>
  );
};

type Resource = { id: string; name: string; systemName: string; status: string };

const ResourceList = ({ title, empty, items, icon: Icon, onDelete, pending }: { title: string; empty: string; items: Resource[]; icon: typeof DatabaseIcon; onDelete: (id: string, name: string) => void; pending: string }) => (
  <section className="overflow-hidden rounded-2xl border border-divider bg-surface"><div className="border-b border-divider px-6 py-5"><h2 className="text-lg font-semibold">{title}</h2></div><div className="divide-y divide-divider">{items.length ? items.map((item) => <div key={item.id} className="flex items-center justify-between gap-4 px-6 py-4"><div className="flex items-center gap-3"><Icon className="size-4 text-foreground-400" /><div><div className="font-medium">{item.name}</div><div className="font-mono text-xs text-foreground-400">{item.systemName}</div></div></div><div className="flex items-center gap-2"><span className={cn("rounded-full px-2 py-1 text-xs", statusClass(item.status))}>{item.status}</span><Button isIconOnly size="sm" variant="danger-soft" aria-label={`Delete ${item.name}`} isDisabled={pending === item.id || item.status === "pending"} onPress={() => onDelete(item.id, item.name)}><Trash2 className="size-4" /></Button></div></div>) : <div className="px-6 py-10 text-center text-sm text-foreground-400">{empty}</div>}</div></section>
);
