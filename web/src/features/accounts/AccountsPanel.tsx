import { Button, Input, Label, TextField, toast } from "@heroui/react";
import { Plus, Power, Trash2, UserRound } from "lucide-react";
import { type SubmitEvent, useState } from "react";
import useSWR, { useSWRConfig } from "swr";
import * as v from "valibot";
import type {
  AccountJobResponse,
  AccountListResponse,
} from "../../api/types";
import { Confirm } from "../../components/Confirm";
import { api, fetcher } from "../../lib/api";
import { cn } from "../../utils/classnames";
import { formatDate } from "../../utils/date";
import { errorMessage } from "../../utils/errors";
import { statusClass } from "../../utils/status";
import { issueMessage, nameField } from "../../utils/validation";

type Props = {
  nodeId: string;
};

export const AccountsPanel = ({ nodeId }: Props) => {
  const [name, setName] = useState("");
  const [pending, setPending] = useState(false);
  const [actionId, setActionId] = useState("");

  const { mutate: mutateKey } = useSWRConfig();
  const { data, mutate } = useSWR<AccountListResponse>("accounts", fetcher);

  const submit = async (event: SubmitEvent<HTMLFormElement>) => {
    event.preventDefault();

    const result = v.safeParse(nameField, name);
    if (!result.success) {
      toast.warning("Check the account name", {
        description: issueMessage(result.issues),
      });
      return;
    }

    setPending(true);

    try {
      await api
        .post("accounts", {
          json: {
            name: result.output,
            nodeId
          }
        })
        .json<AccountJobResponse>();
      setName("");
      await Promise.all([mutate(), mutateKey("jobs")]);
      toast.success("Account queued for creation");
    } catch (requestError) {
      toast.danger("Account action failed", {
        description: await errorMessage(requestError),
      });
    } finally {
      setPending(false);
    }
  };

  const runAction = async (id: string, enabled?: boolean) => {
    setActionId(id);

    try {
      const path = `accounts/${encodeURIComponent(id)}`;

      if (enabled === undefined) {
        await api.delete(path).json<AccountJobResponse>();
      } else {
        await api.patch(path, { json: { enabled } }).json<AccountJobResponse>();
      }
      await Promise.all([mutate(), mutateKey("jobs")]);
      toast.success(
        enabled === undefined
          ? "Account queued for deletion"
          : enabled
            ? "Account enabled"
            : "Account disabled",
      );
    } catch (requestError) {
      toast.danger("Account action failed", {
        description: await errorMessage(requestError),
      });
    } finally {
      setActionId("");
    }
  };

  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_22rem]">
      <section className="panel-card overflow-hidden">
        <div className="border-b border-divider px-6 py-5">
          <h2 className="text-base font-semibold">Hosting accounts</h2>
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
                  <div className="icon-box">
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
                      onPress={() => void runAction(account.id, !account.enabled)}
                    >
                      <Power className="size-4" aria-hidden="true" />
                    </Button>
                    <Confirm
                      title={`Delete ${account.name}?`}
                      description="The account must be empty. Its home directory will be moved to the recovery trash."
                      action="Delete account"
                      onConfirm={() =>
                        void runAction(account.id)
                      }
                    >
                      <Button
                        isIconOnly
                        size="sm"
                        variant="danger-soft"
                        aria-label={`Delete ${account.name}`}
                        isDisabled={
                          account.status === "pending" ||
                          actionId === account.id
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
              No hosting accounts yet.
            </div>
          )}
        </div>
      </section>

      <aside className="panel-card h-fit p-6">
        <h2 className="text-base font-semibold">Create account</h2>
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
