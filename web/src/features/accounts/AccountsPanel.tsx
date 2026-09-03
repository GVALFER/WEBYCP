import { Button, Input, Label, TextField } from "@heroui/react";
import { Plus, Power, Trash2, UserRound } from "lucide-react";
import { type FormEvent, useState } from "react";
import useSWR, { useSWRConfig } from "swr";

import {
  createAccount,
  deleteAccount,
  fetcher,
  setAccount,
  type AccountListResponse,
  type AuthResponse,
} from "../../lib/api";
import { cn } from "../../utils/classnames";
import { formatDate } from "../../utils/date";
import { errorMessage } from "../../utils/errors";
import { statusClass } from "../../utils/status";

type Props = {
  nodeId: string;
  session: AuthResponse;
};

export const AccountsPanel = ({ nodeId, session }: Props) => {
  const [name, setName] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const [actionId, setActionId] = useState("");
  const { mutate: mutateKey } = useSWRConfig();
  const { data, mutate } = useSWR<AccountListResponse>("accounts", fetcher, {
    refreshInterval: 2_000,
  });

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setPending(true);
    setError("");
    try {
      await createAccount({ name, nodeId }, session.csrfToken);
      setName("");
      await Promise.all([mutate(), mutateKey("jobs")]);
    } catch (requestError) {
      setError(await errorMessage(requestError));
    } finally {
      setPending(false);
    }
  };

  const runAction = async (id: string, name: string, enabled?: boolean) => {
    if (enabled === undefined && !window.confirm(`Delete ${name}? The account must be empty and its home will be quarantined.`)) return;
    setActionId(id);
    setError("");
    try {
      if (enabled === undefined) await deleteAccount(id, session.csrfToken);
      else await setAccount(id, enabled, session.csrfToken);
      await Promise.all([mutate(), mutateKey("jobs")]);
    } catch (requestError) {
      setError(await errorMessage(requestError));
    } finally {
      setActionId("");
    }
  };

  return (
    <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_22rem]">
      <section className="overflow-hidden rounded-2xl border border-divider bg-surface">
        <div className="border-b border-divider px-6 py-5">
          <h2 className="text-lg font-semibold">Hosting accounts</h2>
          <div className="mt-1 text-sm text-foreground-500">
            Isolated Linux identities owned by panel users.
          </div>
        </div>
        <div className="divide-y divide-divider">
          {data?.items.length ? (
            data.items.map((account) => (
              <div
                key={account.id}
                className="flex flex-col gap-3 px-6 py-5 sm:flex-row sm:items-center sm:justify-between"
              >
                <div className="flex items-center gap-4">
                  <div className="flex size-10 items-center justify-center rounded-xl bg-default/10">
                    <UserRound className="size-5" aria-hidden="true" />
                  </div>
                  <div>
                    <div className="font-medium">{account.name}</div>
                    <div className="mt-1 font-mono text-xs text-foreground-400">
                      {account.systemUser}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-4">
                  <div className="hidden text-xs text-foreground-400 sm:block">
                    {formatDate(account.createdAt)}
                  </div>
                  <span
                    className={cn(
                      "rounded-full px-2.5 py-1 text-xs capitalize",
                      statusClass(account.status),
                    )}
                  >
                    {account.status}
                  </span>
                  <div className="flex items-center gap-1">
                    <Button
                      isIconOnly
                      size="sm"
                      variant="tertiary"
                      aria-label={account.enabled ? `Disable ${account.name}` : `Enable ${account.name}`}
                      isDisabled={account.status === "pending" || actionId === account.id}
                      onPress={() => void runAction(account.id, account.name, !account.enabled)}
                    >
                      <Power className="size-4" aria-hidden="true" />
                    </Button>
                    <Button
                      isIconOnly
                      size="sm"
                      variant="danger-soft"
                      aria-label={`Delete ${account.name}`}
                      isDisabled={account.status === "pending" || actionId === account.id}
                      onPress={() => void runAction(account.id, account.name)}
                    >
                      <Trash2 className="size-4" aria-hidden="true" />
                    </Button>
                  </div>
                </div>
              </div>
            ))
          ) : (
            <div className="px-6 py-12 text-center text-sm text-foreground-400">
              No hosting accounts yet.
            </div>
          )}
        </div>
      </section>

      <aside className="h-fit rounded-2xl border border-divider bg-surface p-6">
        <h2 className="text-lg font-semibold">Create account</h2>
        <div className="mt-1 text-sm leading-6 text-foreground-500">
          This queues an isolated Linux user on the selected node.
        </div>
        <form className="mt-6 space-y-5" onSubmit={submit}>
          <TextField fullWidth isRequired>
            <Label>Account name</Label>
            <Input
              value={name}
              minLength={2}
              maxLength={80}
              onChange={(event) => setName(event.currentTarget.value)}
            />
          </TextField>
          {error && (
            <div className="rounded-xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">
              {error}
            </div>
          )}
          <Button
            type="submit"
            variant="primary"
            fullWidth
            isDisabled={pending || !nodeId}
          >
            <Plus className="size-4" aria-hidden="true" />
            {pending ? "Queuing…" : "Create account"}
          </Button>
        </form>
      </aside>
    </div>
  );
};
