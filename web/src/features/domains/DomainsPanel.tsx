import { Button, Input, Label, TextField } from "@heroui/react";
import {
  CornerDownRight,
  Globe2,
  Pencil,
  Plus,
  Power,
  Trash2,
} from "lucide-react";
import { type FormEvent, useState } from "react";
import useSWR, { useSWRConfig } from "swr";

import {
  createDomain,
  createDomainAlias,
  deleteDomain,
  deleteDomainAlias,
  fetcher,
  renameDomain,
  renameDomainAlias,
  setDomain,
  setDomainAlias,
  type AccountListResponse,
  type AuthResponse,
  type DomainAliasListResponse,
  type DomainListResponse,
} from "../../lib/api";
import { cn } from "../../utils/classnames";
import { formatDate } from "../../utils/date";
import { errorMessage } from "../../utils/errors";
import { statusClass } from "../../utils/status";

type Props = {
  session: AuthResponse;
};

export const DomainsPanel = ({ session }: Props) => {
  const [accountId, setAccountId] = useState("");
  const [domainName, setDomainName] = useState("");
  const [domainPending, setDomainPending] = useState(false);
  const [domainError, setDomainError] = useState("");
  const [domainId, setDomainId] = useState("");
  const [aliasName, setAliasName] = useState("");
  const [aliasPending, setAliasPending] = useState(false);
  const [aliasError, setAliasError] = useState("");
  const [actionId, setActionId] = useState("");
  const [actionError, setActionError] = useState("");
  const { mutate: mutateKey } = useSWRConfig();
  const { data: accounts } = useSWR<AccountListResponse>("accounts", fetcher, {
    refreshInterval: 2_000,
  });
  const { data: domains, mutate: mutateDomains } =
    useSWR<DomainListResponse>("domains", fetcher, { refreshInterval: 2_000 });
  const activeAccounts = accounts?.items.filter(
    (account) => account.status === "active",
  );
  const activeDomains = domains?.items.filter(
    (domain) => domain.status === "active",
  );
  const selectedAccount = accountId || activeAccounts?.[0]?.id || "";
  const selectedDomain =
    activeDomains?.some((domain) => domain.id === domainId)
      ? domainId
      : activeDomains?.[0]?.id || "";
  const aliasKey = selectedDomain
    ? `domains/${encodeURIComponent(selectedDomain)}/aliases`
    : null;
  const { data: aliases, mutate: mutateAliases } =
    useSWR<DomainAliasListResponse>(aliasKey, fetcher, {
      refreshInterval: 2_000,
    });
  const accountNames = new Map(
    accounts?.items.map((account) => [account.id, account.name]),
  );
  const currentDomain = domains?.items.find(
    (domain) => domain.id === selectedDomain,
  );

  const submitDomain = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!selectedAccount) return;
    setDomainPending(true);
    setDomainError("");
    try {
      await createDomain(
        { accountId: selectedAccount, name: domainName },
        session.csrfToken,
      );
      setDomainName("");
      await Promise.all([mutateDomains(), mutateKey("jobs")]);
    } catch (requestError) {
      setDomainError(await errorMessage(requestError));
    } finally {
      setDomainPending(false);
    }
  };

  const submitAlias = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!selectedDomain) return;
    setAliasPending(true);
    setAliasError("");
    try {
      await createDomainAlias(
        selectedDomain,
        { name: aliasName },
        session.csrfToken,
      );
      setAliasName("");
      await Promise.all([mutateAliases(), mutateKey("jobs")]);
    } catch (requestError) {
      setAliasError(await errorMessage(requestError));
    } finally {
      setAliasPending(false);
    }
  };

  const runDomainAction = async (id: string, name: string, enabled?: boolean) => {
    if (
      enabled === undefined &&
      !window.confirm(
        `Delete ${name}? Its document root will be moved to the recovery trash.`,
      )
    ) {
      return;
    }
    await runAction(`domain:${id}`, false, async () => {
      if (enabled === undefined) {
        await deleteDomain(id, session.csrfToken);
      } else {
        await setDomain(id, enabled, session.csrfToken);
      }
    });
  };

  const runAliasAction = async (id: string, name: string, enabled?: boolean) => {
    if (!selectedDomain) return;
    if (enabled === undefined && !window.confirm(`Delete alias ${name}?`)) return;
    await runAction(`alias:${id}`, true, async () => {
      if (enabled === undefined) {
        await deleteDomainAlias(selectedDomain, id, session.csrfToken);
      } else {
        await setDomainAlias(selectedDomain, id, enabled, session.csrfToken);
      }
    });
  };

  const runAction = async (
    id: string,
    alias: boolean,
    action: () => Promise<void>,
  ) => {
    setActionId(id);
    setActionError("");
    try {
      await action();
      const refreshes = [mutateDomains(), mutateKey("jobs")];
      if (alias) refreshes.push(mutateAliases());
      await Promise.all(refreshes);
    } catch (requestError) {
      setActionError(await errorMessage(requestError));
    } finally {
      setActionId("");
    }
  };

  const renamePrimary = async (id: string, current: string) => {
    const name = window.prompt("New domain name", current)?.trim();
    if (!name || name === current) return;
    await runAction(`domain:${id}`, false, async () => {
      await renameDomain(id, name, session.csrfToken);
    });
  };

  const renameAlias = async (id: string, current: string) => {
    if (!selectedDomain) return;
    const name = window.prompt("New alias name", current)?.trim();
    if (!name || name === current) return;
    await runAction(`alias:${id}`, true, async () => {
      await renameDomainAlias(selectedDomain, id, name, session.csrfToken);
    });
  };

  return (
    <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_22rem]">
      <div className="space-y-6">
        <section className="overflow-hidden rounded-2xl border border-divider bg-surface">
          <div className="border-b border-divider px-6 py-5">
            <h2 className="text-lg font-semibold">Domains</h2>
            <div className="mt-1 text-sm text-foreground-500">
              Nginx sites and isolated document roots.
            </div>
            {actionError && (
              <div className="mt-3 rounded-xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">
                {actionError}
              </div>
            )}
          </div>
          <div className="divide-y divide-divider">
            {domains?.items.length ? (
              domains.items.map((domain) => (
                <div
                  key={domain.id}
                  className="flex flex-col gap-3 px-6 py-5 sm:flex-row sm:items-center sm:justify-between"
                >
                  <div className="flex items-center gap-4">
                    <div className="flex size-10 items-center justify-center rounded-xl bg-default/10">
                      <Globe2 className="size-5" aria-hidden="true" />
                    </div>
                    <div>
                      <div className="font-medium">{domain.name}</div>
                      <div className="mt-1 text-xs text-foreground-400">
                        {accountNames.get(domain.accountId) ?? domain.accountId} ·
                        PHP {domain.phpVersion}
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    <div className="hidden text-xs text-foreground-400 sm:block">
                      {formatDate(domain.createdAt)}
                    </div>
                    <span
                      className={cn(
                        "rounded-full px-2.5 py-1 text-xs capitalize",
                        statusClass(domain.status),
                      )}
                    >
                      {domain.status}
                    </span>
                    <div className="flex items-center gap-1">
                      <Button
                        isIconOnly
                        size="sm"
                        variant="tertiary"
                        aria-label={`Rename ${domain.name}`}
                        isDisabled={domain.status !== "active" || actionId === `domain:${domain.id}`}
                        onPress={() => renamePrimary(domain.id, domain.name)}
                      >
                        <Pencil className="size-4" aria-hidden="true" />
                      </Button>
                      <Button
                        isIconOnly
                        size="sm"
                        variant="tertiary"
                        aria-label={domain.enabled ? `Disable ${domain.name}` : `Enable ${domain.name}`}
                        isDisabled={domain.status === "pending" || actionId === `domain:${domain.id}`}
                        onPress={() => runDomainAction(domain.id, domain.name, !domain.enabled)}
                      >
                        <Power className="size-4" aria-hidden="true" />
                      </Button>
                      <Button
                        isIconOnly
                        size="sm"
                        variant="danger-soft"
                        aria-label={`Delete ${domain.name}`}
                        isDisabled={domain.status === "pending" || actionId === `domain:${domain.id}`}
                        onPress={() => runDomainAction(domain.id, domain.name)}
                      >
                        <Trash2 className="size-4" aria-hidden="true" />
                      </Button>
                    </div>
                  </div>
                </div>
              ))
            ) : (
              <div className="px-6 py-12 text-center text-sm text-foreground-400">
                No domains yet.
              </div>
            )}
          </div>
        </section>

        <section className="overflow-hidden rounded-2xl border border-divider bg-surface">
          <div className="border-b border-divider px-6 py-5">
            <h2 className="text-lg font-semibold">Aliases</h2>
            <div className="mt-1 text-sm text-foreground-500">
              {currentDomain
                ? `Names serving the ${currentDomain.name} document root.`
                : "Select an active domain to manage its aliases."}
            </div>
          </div>
          <div className="divide-y divide-divider">
            {aliases?.items.length ? (
              aliases.items.map((alias) => (
                <div
                  key={alias.id}
                  className="flex items-center justify-between gap-4 px-6 py-4"
                >
                  <div className="flex items-center gap-3">
                    <CornerDownRight
                      className="size-4 text-foreground-400"
                      aria-hidden="true"
                    />
                    <div className="font-medium">{alias.name}</div>
                  </div>
                  <div className="flex items-center gap-2">
                    <span
                      className={cn(
                        "rounded-full px-2.5 py-1 text-xs capitalize",
                        statusClass(alias.status),
                      )}
                    >
                      {alias.status}
                    </span>
                    <Button
                      isIconOnly
                      size="sm"
                      variant="tertiary"
                      aria-label={`Rename ${alias.name}`}
                      isDisabled={alias.status === "pending" || actionId === `alias:${alias.id}`}
                      onPress={() => renameAlias(alias.id, alias.name)}
                    >
                      <Pencil className="size-4" aria-hidden="true" />
                    </Button>
                    <Button
                      isIconOnly
                      size="sm"
                      variant="tertiary"
                      aria-label={alias.enabled ? `Disable ${alias.name}` : `Enable ${alias.name}`}
                      isDisabled={alias.status === "pending" || actionId === `alias:${alias.id}`}
                      onPress={() => runAliasAction(alias.id, alias.name, !alias.enabled)}
                    >
                      <Power className="size-4" aria-hidden="true" />
                    </Button>
                    <Button
                      isIconOnly
                      size="sm"
                      variant="danger-soft"
                      aria-label={`Delete ${alias.name}`}
                      isDisabled={alias.status === "pending" || actionId === `alias:${alias.id}`}
                      onPress={() => runAliasAction(alias.id, alias.name)}
                    >
                      <Trash2 className="size-4" aria-hidden="true" />
                    </Button>
                  </div>
                </div>
              ))
            ) : (
              <div className="px-6 py-10 text-center text-sm text-foreground-400">
                {selectedDomain ? "No aliases yet." : "No active domain selected."}
              </div>
            )}
          </div>
        </section>
      </div>

      <div className="space-y-6">
        <aside className="rounded-2xl border border-divider bg-surface p-6">
          <h2 className="text-lg font-semibold">Add domain</h2>
          <div className="mt-1 text-sm leading-6 text-foreground-500">
            Creates the document root and validates services before reloading.
          </div>
          <form className="mt-6 space-y-5" onSubmit={submitDomain}>
            <label className="block space-y-2">
              <span className="text-sm font-medium">Hosting account</span>
              <select
                required
                value={selectedAccount}
                onChange={(event) => setAccountId(event.currentTarget.value)}
                className="h-10 w-full rounded-lg border border-divider bg-background px-3 text-sm outline-none focus:border-accent"
              >
                {!activeAccounts?.length && (
                  <option value="">No active accounts</option>
                )}
                {activeAccounts?.map((account) => (
                  <option key={account.id} value={account.id}>
                    {account.name}
                  </option>
                ))}
              </select>
            </label>
            <TextField fullWidth isRequired>
              <Label>Domain name</Label>
              <Input
                value={domainName}
                placeholder="example.com"
                autoCapitalize="none"
                autoCorrect="off"
                minLength={4}
                maxLength={253}
                onChange={(event) => setDomainName(event.currentTarget.value)}
              />
            </TextField>
            {domainError && (
              <div className="rounded-xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">
                {domainError}
              </div>
            )}
            <Button
              type="submit"
              variant="primary"
              fullWidth
              isDisabled={domainPending || !selectedAccount}
            >
              <Plus className="size-4" aria-hidden="true" />
              {domainPending ? "Queuing…" : "Add domain"}
            </Button>
          </form>
        </aside>

        <aside className="rounded-2xl border border-divider bg-surface p-6">
          <h2 className="text-lg font-semibold">Add alias</h2>
          <div className="mt-1 text-sm leading-6 text-foreground-500">
            Points another hostname at an existing domain.
          </div>
          <form className="mt-6 space-y-5" onSubmit={submitAlias}>
            <label className="block space-y-2">
              <span className="text-sm font-medium">Primary domain</span>
              <select
                required
                value={selectedDomain}
                onChange={(event) => setDomainId(event.currentTarget.value)}
                className="h-10 w-full rounded-lg border border-divider bg-background px-3 text-sm outline-none focus:border-accent"
              >
                {!activeDomains?.length && (
                  <option value="">No active domains</option>
                )}
                {activeDomains?.map((domain) => (
                  <option key={domain.id} value={domain.id}>
                    {domain.name}
                  </option>
                ))}
              </select>
            </label>
            <TextField fullWidth isRequired>
              <Label>Alias name</Label>
              <Input
                value={aliasName}
                placeholder="www.example.com"
                autoCapitalize="none"
                autoCorrect="off"
                minLength={4}
                maxLength={253}
                onChange={(event) => setAliasName(event.currentTarget.value)}
              />
            </TextField>
            {aliasError && (
              <div className="rounded-xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">
                {aliasError}
              </div>
            )}
            <Button
              type="submit"
              variant="primary"
              fullWidth
              isDisabled={aliasPending || !selectedDomain}
            >
              <Plus className="size-4" aria-hidden="true" />
              {aliasPending ? "Queuing…" : "Add alias"}
            </Button>
          </form>
        </aside>
      </div>
    </div>
  );
};
