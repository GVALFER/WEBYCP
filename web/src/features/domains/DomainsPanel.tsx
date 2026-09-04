import { Button, Input, Label, TextField, toast } from "@heroui/react";
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
import * as v from "valibot";

import type {
  AccountListResponse,
  DomainAliasJobResponse,
  DomainAliasListResponse,
  DomainJobResponse,
  DomainListResponse,
} from "../../api/types";
import { Confirm } from "../../components/Confirm";
import { SelectField } from "../../components/SelectField";
import { TextDialog } from "../../components/TextDialog";
import { api, fetcher } from "../../lib/api";
import { cn } from "../../utils/classnames";
import { formatDate } from "../../utils/date";
import { errorMessage } from "../../utils/errors";
import { statusClass } from "../../utils/status";
import { domainField, issueMessage } from "../../utils/validation";

export const DomainsPanel = () => {
  const [accountId, setAccountId] = useState("");
  const [domainName, setDomainName] = useState("");
  const [domainPending, setDomainPending] = useState(false);
  const [domainId, setDomainId] = useState("");
  const [aliasName, setAliasName] = useState("");
  const [aliasPending, setAliasPending] = useState(false);
  const [actionId, setActionId] = useState("");
  const [edit, setEdit] = useState<{
    kind: "domain" | "alias";
    id: string;
    name: string;
  } | null>(null);
  const { mutate: mutateKey } = useSWRConfig();
  const { data: accounts } = useSWR<AccountListResponse>("accounts", fetcher);
  const { data: domains, mutate: mutateDomains } =
    useSWR<DomainListResponse>("domains", fetcher);
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
    useSWR<DomainAliasListResponse>(aliasKey, fetcher);
  const accountNames = new Map(
    accounts?.items.map((account) => [account.id, account.name]),
  );
  const currentDomain = domains?.items.find(
    (domain) => domain.id === selectedDomain,
  );

  const submitDomain = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!selectedAccount) return;
    const result = v.safeParse(domainField, domainName);
    if (!result.success) {
      toast.warning("Check the domain name", {
        description: issueMessage(result.issues),
      });
      return;
    }
    setDomainPending(true);
    try {
      await api
        .post("domains", {
          json: { accountId: selectedAccount, name: result.output },
        })
        .json<DomainJobResponse>();
      setDomainName("");
      await Promise.all([mutateDomains(), mutateKey("jobs")]);
      toast.success("Domain queued for creation");
    } catch (requestError) {
      toast.danger("Domain action failed", {
        description: await errorMessage(requestError),
      });
    } finally {
      setDomainPending(false);
    }
  };

  const submitAlias = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!selectedDomain) return;
    const result = v.safeParse(domainField, aliasName);
    if (!result.success) {
      toast.warning("Check the alias name", {
        description: issueMessage(result.issues),
      });
      return;
    }
    setAliasPending(true);
    try {
      await api
        .post(`domains/${encodeURIComponent(selectedDomain)}/aliases`, {
          json: { name: result.output },
        })
        .json<DomainAliasJobResponse>();
      setAliasName("");
      await Promise.all([mutateAliases(), mutateKey("jobs")]);
      toast.success("Alias queued for creation");
    } catch (requestError) {
      toast.danger("Alias action failed", {
        description: await errorMessage(requestError),
      });
    } finally {
      setAliasPending(false);
    }
  };

  const runDomainAction = async (id: string, enabled?: boolean) => {
    await runAction(`domain:${id}`, false, async () => {
      const path = `domains/${encodeURIComponent(id)}`;
      if (enabled === undefined) {
        await api.delete(path).json<DomainJobResponse>();
      } else {
        await api.patch(path, { json: { enabled } }).json<DomainJobResponse>();
      }
    }, enabled === undefined ? "Domain queued for deletion" : enabled ? "Domain enabled" : "Domain disabled");
  };

  const runAliasAction = async (id: string, enabled?: boolean) => {
    if (!selectedDomain) return;
    await runAction(`alias:${id}`, true, async () => {
      const path = `domains/${encodeURIComponent(selectedDomain)}/aliases/${encodeURIComponent(id)}`;
      if (enabled === undefined) {
        await api.delete(path).json<DomainAliasJobResponse>();
      } else {
        await api
          .patch(path, { json: { enabled } })
          .json<DomainAliasJobResponse>();
      }
    }, enabled === undefined ? "Alias queued for deletion" : enabled ? "Alias enabled" : "Alias disabled");
  };

  const runAction = async (
    id: string,
    alias: boolean,
    action: () => Promise<void>,
    success: string,
  ) => {
    setActionId(id);
    try {
      await action();
      const refreshes = [mutateDomains(), mutateKey("jobs")];
      if (alias) refreshes.push(mutateAliases());
      await Promise.all(refreshes);
      toast.success(success);
    } catch (requestError) {
      toast.danger("Domain action failed", {
        description: await errorMessage(requestError),
      });
    } finally {
      setActionId("");
    }
  };

  const rename = async (name: string) => {
    if (!edit || name === edit.name) {
      setEdit(null);
      return;
    }
    const result = v.safeParse(domainField, name);
    if (!result.success) {
      toast.warning("Check the hostname", {
        description: issueMessage(result.issues),
      });
      return;
    }
    const editing = edit;
    const isAlias = editing.kind === "alias";
    if (isAlias && !selectedDomain) return;
    const path = isAlias
      ? `domains/${encodeURIComponent(selectedDomain)}/aliases/${encodeURIComponent(editing.id)}`
      : `domains/${encodeURIComponent(editing.id)}`;
    await runAction(
      `${editing.kind}:${editing.id}`,
      isAlias,
      async () => {
        if (isAlias) {
          await api
            .patch(path, { json: { name: result.output } })
            .json<DomainAliasJobResponse>();
        } else {
          await api
            .patch(path, { json: { name: result.output } })
            .json<DomainJobResponse>();
        }
      },
      isAlias ? "Alias rename queued" : "Domain rename queued",
    );
    setEdit(null);
  };

  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_22rem]">
      <div className="space-y-6">
        <section className="panel-card overflow-hidden">
          <div className="border-b border-divider px-6 py-5">
            <h2 className="text-base font-semibold">Domains</h2>
            <div className="mt-1 text-sm text-foreground-500">
              Nginx sites and isolated document roots.
            </div>
          </div>
          <div className="divide-y divide-divider">
            {domains?.items.length ? (
              domains.items.map((domain) => (
                <div
                  key={domain.id}
                  className="flex flex-col gap-3 px-6 py-5 sm:flex-row sm:items-center sm:justify-between"
                >
                  <div className="flex items-center gap-4">
                    <div className="icon-box">
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
                        onPress={() =>
                          setEdit({
                            kind: "domain",
                            id: domain.id,
                            name: domain.name,
                          })
                        }
                      >
                        <Pencil className="size-4" aria-hidden="true" />
                      </Button>
                      <Button
                        isIconOnly
                        size="sm"
                        variant="tertiary"
                        aria-label={domain.enabled ? `Disable ${domain.name}` : `Enable ${domain.name}`}
                        isDisabled={domain.status === "pending" || actionId === `domain:${domain.id}`}
                        onPress={() =>
                          void runDomainAction(domain.id, !domain.enabled)
                        }
                      >
                        <Power className="size-4" aria-hidden="true" />
                      </Button>
                      <Confirm
                        title={`Delete ${domain.name}?`}
                        description="Its document root will be moved to the recovery trash."
                        action="Delete domain"
                        onConfirm={() => void runDomainAction(domain.id)}
                      >
                        <Button
                          isIconOnly
                          size="sm"
                          variant="danger-soft"
                          aria-label={`Delete ${domain.name}`}
                          isDisabled={
                            domain.status === "pending" ||
                            actionId === `domain:${domain.id}`
                          }
                        >
                          <Trash2 className="size-4" aria-hidden="true" />
                        </Button>
                      </Confirm>
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

        <section className="panel-card overflow-hidden">
          <div className="border-b border-divider px-6 py-5">
            <h2 className="text-base font-semibold">Aliases</h2>
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
                      onPress={() =>
                        setEdit({
                          kind: "alias",
                          id: alias.id,
                          name: alias.name,
                        })
                      }
                    >
                      <Pencil className="size-4" aria-hidden="true" />
                    </Button>
                    <Button
                      isIconOnly
                      size="sm"
                      variant="tertiary"
                      aria-label={alias.enabled ? `Disable ${alias.name}` : `Enable ${alias.name}`}
                      isDisabled={alias.status === "pending" || actionId === `alias:${alias.id}`}
                      onPress={() =>
                        void runAliasAction(alias.id, !alias.enabled)
                      }
                    >
                      <Power className="size-4" aria-hidden="true" />
                    </Button>
                    <Confirm
                      title={`Delete ${alias.name}?`}
                      description="This hostname will stop serving the primary domain."
                      action="Delete alias"
                      onConfirm={() => void runAliasAction(alias.id)}
                    >
                      <Button
                        isIconOnly
                        size="sm"
                        variant="danger-soft"
                        aria-label={`Delete ${alias.name}`}
                        isDisabled={
                          alias.status === "pending" ||
                          actionId === `alias:${alias.id}`
                        }
                      >
                        <Trash2 className="size-4" aria-hidden="true" />
                      </Button>
                    </Confirm>
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
        <aside className="panel-card p-6">
          <h2 className="text-base font-semibold">Add domain</h2>
          <div className="mt-1 text-sm leading-6 text-foreground-500">
            Creates the document root and validates services before reloading.
          </div>
          <form className="mt-6 space-y-5" onSubmit={submitDomain}>
            <SelectField
              required
              label="Hosting account"
              value={selectedAccount}
              options={activeAccounts ?? []}
              empty="No active accounts"
              onChange={setAccountId}
            />
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

        <aside className="panel-card p-6">
          <h2 className="text-base font-semibold">Add alias</h2>
          <div className="mt-1 text-sm leading-6 text-foreground-500">
            Points another hostname at an existing domain.
          </div>
          <form className="mt-6 space-y-5" onSubmit={submitAlias}>
            <SelectField
              required
              label="Primary domain"
              value={selectedDomain}
              options={activeDomains ?? []}
              empty="No active domains"
              onChange={setDomainId}
            />
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
      <TextDialog
        open={edit !== null}
        title={edit?.kind === "alias" ? "Rename alias" : "Rename domain"}
        label="Hostname"
        value={edit?.name ?? ""}
        pending={actionId !== ""}
        onOpenChange={(open) => {
          if (!open) setEdit(null);
        }}
        onSubmit={(value) => void rename(value)}
      />
    </div>
  );
};
