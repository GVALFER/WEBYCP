import { Button, Input, Label, TextField, toast } from "@heroui/react";
import {
  Database as DatabaseIcon,
  KeyRound,
  Plus,
  Trash2,
  UserRound,
} from "lucide-react";
import { type FormEvent, useState } from "react";
import useSWR, { useSWRConfig } from "swr";
import * as v from "valibot";

import type {
  AccountListResponse,
  DatabaseGrantJobResponse,
  DatabaseGrantListResponse,
  DatabaseJobResponse,
  DatabaseListResponse,
  DatabaseUserJobResponse,
  DatabaseUserListResponse,
} from "../../api/types";
import { Confirm } from "../../components/Confirm";
import { SelectField } from "../../components/SelectField";
import { api, fetcher } from "../../lib/api";
import { cn } from "../../utils/classnames";
import { errorMessage } from "../../utils/errors";
import { statusClass } from "../../utils/status";
import { dbNameField, issueMessage } from "../../utils/validation";

export const DatabasesPanel = () => {
  const [accountId, setAccountId] = useState("");
  const [name, setName] = useState("");
  const [userName, setUserName] = useState("");
  const [databaseId, setDatabaseId] = useState("");
  const [userId, setUserId] = useState("");
  const [password, setPassword] = useState("");
  const [pending, setPending] = useState("");
  const { mutate: mutateKey } = useSWRConfig();
  const { data: accounts } = useSWR<AccountListResponse>("accounts", fetcher);
  const { data: databases, mutate: mutateDatabases } =
    useSWR<DatabaseListResponse>("databases", fetcher);
  const { data: users, mutate: mutateUsers } =
    useSWR<DatabaseUserListResponse>("database-users", fetcher);
  const { data: grants, mutate: mutateGrants } =
    useSWR<DatabaseGrantListResponse>("database-grants", fetcher);
  const activeAccounts =
    accounts?.items.filter(
      (account) => account.status === "active" && account.enabled,
    ) ?? [];
  const selectedAccount = accountId || activeAccounts[0]?.id || "";
  const accountDatabases =
    databases?.items.filter(
      (item) =>
        item.accountId === selectedAccount && item.status === "active",
    ) ?? [];
  const accountUsers =
    users?.items.filter(
      (item) =>
        item.accountId === selectedAccount && item.status === "active",
    ) ?? [];
  const selectedDatabase = databaseId || accountDatabases[0]?.id || "";
  const selectedUser = userId || accountUsers[0]?.id || "";

  const refresh = () =>
    Promise.all([
      mutateDatabases(),
      mutateUsers(),
      mutateGrants(),
      mutateKey("jobs"),
    ]);

  const run = async (
    key: string,
    action: () => Promise<unknown>,
    success: string,
  ) => {
    setPending(key);
    try {
      await action();
      await refresh();
      toast.success(success);
    } catch (error) {
      toast.danger("Action failed", {
        description: await errorMessage(error),
      });
    } finally {
      setPending("");
    }
  };

  const submitDatabase = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const result = v.safeParse(dbNameField, name);
    if (!result.success) {
      toast.warning("Check the database name", {
        description: issueMessage(result.issues),
      });
      return;
    }
    await run(
      "database",
      () =>
        api
          .post("databases", {
            json: { accountId: selectedAccount, name: result.output },
          })
          .json<DatabaseJobResponse>(),
      "Database queued for creation",
    );
    setName("");
  };

  const submitUser = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const result = v.safeParse(dbNameField, userName);
    if (!result.success) {
      toast.warning("Check the user name", {
        description: issueMessage(result.issues),
      });
      return;
    }
    await run(
      "user",
      async () => {
        const response = await api
          .post("database-users", {
            json: { accountId: selectedAccount, name: result.output },
          })
          .json<DatabaseUserJobResponse>();
        setPassword(response.password ?? "");
      },
      "Database user queued for creation",
    );
    setUserName("");
  };

  const submitGrant = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!selectedDatabase || !selectedUser) return;
    const path = `databases/${encodeURIComponent(selectedDatabase)}/users/${encodeURIComponent(selectedUser)}`;
    await run(
      "grant",
      () => api.put(path).json<DatabaseGrantJobResponse>(),
      "Access granted",
    );
  };

  const removeDatabase = (id: string) =>
    run(
      id,
      () =>
        api
          .delete(`databases/${encodeURIComponent(id)}`)
          .json<DatabaseJobResponse>(),
      "Database queued for deletion",
    );

  const removeUser = (id: string) =>
    run(
      id,
      () =>
        api
          .delete(`database-users/${encodeURIComponent(id)}`)
          .json<DatabaseUserJobResponse>(),
      "Database user queued for deletion",
    );

  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_22rem]">
      <div className="space-y-6">
        {password && (
          <div className="rounded-2xl border border-warning/30 bg-warning/10 p-5">
            <div className="font-medium">Save this password now</div>
            <div className="mt-3 select-all break-all rounded-xl bg-background/70 px-4 py-3 font-mono text-sm">
              {password}
            </div>
            <div className="mt-2 text-xs text-foreground-500">
              It will not be shown again.
            </div>
          </div>
        )}
        <ResourceList
          title="MySQL databases"
          empty="No databases yet."
          items={databases?.items ?? []}
          icon={DatabaseIcon}
          pending={pending}
          description={(item) =>
            `Delete ${item.name}? Its data will be permanently removed.`
          }
          onDelete={(id) => void removeDatabase(id)}
        />
        <ResourceList
          title="MySQL users"
          empty="No database users yet."
          items={users?.items ?? []}
          icon={UserRound}
          pending={pending}
          description={(item) => `Delete database user ${item.name}?`}
          onDelete={(id) => void removeUser(id)}
        />
        <section className="panel-card p-6">
          <h2 className="text-base font-semibold">Grants</h2>
          <div className="mt-4 space-y-2">
            {grants?.items.length ? (
              grants.items.map((grant) => {
                const database = databases?.items.find(
                  (item) => item.id === grant.databaseId,
                );
                const user = users?.items.find(
                  (item) => item.id === grant.databaseUserId,
                );
                const path = `databases/${encodeURIComponent(grant.databaseId)}/users/${encodeURIComponent(grant.databaseUserId)}`;
                return (
                  <div
                    key={`${grant.databaseId}:${grant.databaseUserId}`}
                    className="flex items-center justify-between rounded-xl border border-border/70 bg-surface-secondary/60 px-4 py-3 text-sm"
                  >
                    <div>
                      <span className="font-medium">{user?.name}</span>
                      <span className="mx-2 text-foreground-400">→</span>
                      {database?.name}
                    </div>
                    <Confirm
                      title="Revoke database access?"
                      description={`${user?.name ?? "This user"} will no longer have access to ${database?.name ?? "this database"}.`}
                      action="Revoke access"
                      onConfirm={() =>
                        void run(
                          `grant:${grant.databaseId}`,
                          () =>
                            api.delete(path).json<DatabaseGrantJobResponse>(),
                          "Access revoked",
                        )
                      }
                    >
                      <Button
                        isIconOnly
                        size="sm"
                        variant="danger-soft"
                        aria-label="Revoke grant"
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </Confirm>
                  </div>
                );
              })
            ) : (
              <div className="py-6 text-center text-sm text-foreground-400">
                No grants yet.
              </div>
            )}
          </div>
        </section>
      </div>

      <aside className="space-y-6">
        <section className="panel-card p-6">
          <h2 className="text-base font-semibold">Create resources</h2>
          <SelectField
            className="mt-5"
            label="Hosting account"
            value={selectedAccount}
            options={activeAccounts}
            onChange={setAccountId}
          />
          <form className="mt-5 space-y-4" onSubmit={submitDatabase}>
            <TextField fullWidth isRequired>
              <Label>Database name</Label>
              <Input
                value={name}
                maxLength={32}
                onChange={(event) => setName(event.currentTarget.value)}
              />
            </TextField>
            <Button
              type="submit"
              variant="primary"
              fullWidth
              isDisabled={!selectedAccount || pending !== ""}
            >
              <Plus className="size-4" />
              Create database
            </Button>
          </form>
          <form
            className="mt-6 space-y-4 border-t border-divider pt-6"
            onSubmit={submitUser}
          >
            <TextField fullWidth isRequired>
              <Label>User name</Label>
              <Input
                value={userName}
                maxLength={32}
                onChange={(event) => setUserName(event.currentTarget.value)}
              />
            </TextField>
            <Button
              type="submit"
              variant="secondary"
              fullWidth
              isDisabled={!selectedAccount || pending !== ""}
            >
              <KeyRound className="size-4" />
              Create user
            </Button>
          </form>
        </section>

        <form className="panel-card p-6" onSubmit={submitGrant}>
          <h2 className="text-base font-semibold">Add grant</h2>
          <SelectField
            className="mt-5"
            label="Database"
            value={selectedDatabase}
            onChange={setDatabaseId}
            options={accountDatabases}
          />
          <SelectField
            className="mt-4"
            label="User"
            value={selectedUser}
            onChange={setUserId}
            options={accountUsers}
          />
          <Button
            className="mt-5"
            type="submit"
            variant="primary"
            fullWidth
            isDisabled={!selectedDatabase || !selectedUser || pending !== ""}
          >
            Grant access
          </Button>
        </form>
      </aside>
    </div>
  );
};

type Resource = {
  id: string;
  name: string;
  systemName: string;
  status: string;
};

type ResourceListProps = {
  title: string;
  empty: string;
  items: Resource[];
  icon: typeof DatabaseIcon;
  pending: string;
  description: (item: Resource) => string;
  onDelete: (id: string) => void;
};

const ResourceList = ({
  title,
  empty,
  items,
  icon: Icon,
  pending,
  description,
  onDelete,
}: ResourceListProps) => (
  <section className="panel-card overflow-hidden">
    <div className="border-b border-divider px-6 py-5">
      <h2 className="text-base font-semibold">{title}</h2>
    </div>
    <div className="divide-y divide-divider">
      {items.length ? (
        items.map((item) => (
          <div
            key={item.id}
            className="flex items-center justify-between gap-4 px-6 py-4"
          >
            <div className="flex min-w-0 items-center gap-3">
              <Icon className="size-4 shrink-0 text-foreground-400" />
              <div className="min-w-0">
                <div className="truncate font-medium">{item.name}</div>
                <div className="truncate font-mono text-xs text-foreground-400">
                  {item.systemName}
                </div>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <span
                className={cn(
                  "rounded-full px-2 py-1 text-xs capitalize",
                  statusClass(item.status),
                )}
              >
                {item.status}
              </span>
              <Confirm
                title={`Delete ${item.name}?`}
                description={description(item)}
                action="Delete"
                onConfirm={() => onDelete(item.id)}
              >
                <Button
                  isIconOnly
                  size="sm"
                  variant="danger-soft"
                  aria-label={`Delete ${item.name}`}
                  isDisabled={pending === item.id || item.status === "pending"}
                >
                  <Trash2 className="size-4" />
                </Button>
              </Confirm>
            </div>
          </div>
        ))
      ) : (
        <div className="px-6 py-10 text-center text-sm text-foreground-400">
          {empty}
        </div>
      )}
    </div>
  </section>
);
