import { Button, Input, Label, TextField, toast } from "@heroui/react";
import { Clock3, Plus, Power, Trash2 } from "lucide-react";
import { type FormEvent, useState } from "react";
import useSWR, { useSWRConfig } from "swr";
import * as v from "valibot";

import type {
  AccountListResponse,
  CronJobResponse,
  CronJobListResponse,
  Job,
} from "../../api/types";
import { Confirm } from "../../components/Confirm";
import { SelectField } from "../../components/SelectField";
import { api, fetcher } from "../../lib/api";
import { cn } from "../../utils/classnames";
import { errorMessage } from "../../utils/errors";
import { statusClass } from "../../utils/status";
import {
  commandField,
  issueMessage,
  nameField,
  scheduleField,
} from "../../utils/validation";

const cronSchema = v.object({
  name: nameField,
  schedule: v.pipe(scheduleField, v.nonEmpty("Enter a schedule.")),
  command: commandField,
});

export const CronPanel = () => {
  const [accountId, setAccountId] = useState("");
  const [name, setName] = useState("");
  const [schedule, setSchedule] = useState("0 * * * *");
  const [command, setCommand] = useState("");
  const [pending, setPending] = useState("");
  const { mutate: mutateKey } = useSWRConfig();
  const { data: accounts } = useSWR<AccountListResponse>("accounts", fetcher);
  const { data, mutate } = useSWR<CronJobListResponse>("cron-jobs", fetcher);
  const activeAccounts =
    accounts?.items.filter(
      (account) => account.status === "active" && account.enabled,
    ) ?? [];
  const selectedAccount = accountId || activeAccounts[0]?.id || "";

  const run = async (
    key: string,
    action: () => Promise<unknown>,
    success: string,
  ) => {
    setPending(key);
    try {
      await action();
      await Promise.all([mutate(), mutateKey("jobs")]);
      toast.success(success);
    } catch (error) {
      toast.danger("Action failed", {
        description: await errorMessage(error),
      });
    } finally {
      setPending("");
    }
  };

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const result = v.safeParse(cronSchema, { name, schedule, command });
    if (!result.success) {
      toast.warning("Check the cron job", {
        description: issueMessage(result.issues),
      });
      return;
    }
    await run(
      "create",
      () =>
        api
          .post("cron-jobs", {
            json: {
              accountId: selectedAccount,
              ...result.output,
              enabled: true,
            },
          })
          .json<CronJobResponse>(),
      "Cron job queued for creation",
    );
    setName("");
    setCommand("");
  };

  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_22rem]">
      <section className="panel-card overflow-hidden">
        <div className="border-b border-divider px-6 py-5">
          <h2 className="text-base font-semibold">Cron jobs</h2>
          <div className="mt-1 text-sm text-foreground-500">
            Commands run as the hosting account from its home directory.
          </div>
        </div>
        <div className="divide-y divide-divider">
          {data?.items.length ? (
            data.items.map((item) => {
              const path = `cron-jobs/${encodeURIComponent(item.id)}`;
              const body = {
                accountId: item.accountId,
                name: item.name,
                schedule: item.schedule,
                command: item.command,
                enabled: !item.enabled,
              };
              return (
                <div
                  key={item.id}
                  className="flex flex-col gap-4 px-6 py-5 sm:flex-row sm:items-center sm:justify-between"
                >
                  <div className="flex min-w-0 items-center gap-4">
                    <div className="icon-box">
                      <Clock3 className="size-5" />
                    </div>
                    <div className="min-w-0">
                      <div className="font-medium">{item.name}</div>
                      <div className="mt-1 truncate font-mono text-xs text-foreground-400">
                        {item.schedule} · {item.command}
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
                    <Button
                      isIconOnly
                      size="sm"
                      variant="tertiary"
                      aria-label={
                        item.enabled
                          ? `Disable ${item.name}`
                          : `Enable ${item.name}`
                      }
                      isDisabled={
                        pending === item.id || item.status === "pending"
                      }
                      onPress={() =>
                        void run(
                          item.id,
                          () =>
                            api.patch(path, { json: body }).json<CronJobResponse>(),
                          item.enabled ? "Cron job disabled" : "Cron job enabled",
                        )
                      }
                    >
                      <Power className="size-4" />
                    </Button>
                    <Confirm
                      title={`Delete ${item.name}?`}
                      description="The schedule will be removed from the hosting account."
                      action="Delete"
                      onConfirm={() =>
                        void run(
                          item.id,
                          () => api.delete(path).json<Job>(),
                          "Cron job queued for deletion",
                        )
                      }
                    >
                      <Button
                        isIconOnly
                        size="sm"
                        variant="danger-soft"
                        aria-label={`Delete ${item.name}`}
                        isDisabled={pending === item.id}
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </Confirm>
                  </div>
                </div>
              );
            })
          ) : (
            <div className="px-6 py-12 text-center text-sm text-foreground-400">
              No cron jobs yet.
            </div>
          )}
        </div>
      </section>

      <form className="panel-card h-fit p-6" onSubmit={submit}>
        <h2 className="text-base font-semibold">Add cron job</h2>
        <SelectField
          className="mt-5"
          label="Hosting account"
          value={selectedAccount}
          options={activeAccounts}
          onChange={setAccountId}
        />
        <TextField className="mt-4" fullWidth isRequired>
          <Label>Name</Label>
          <Input
            value={name}
            maxLength={80}
            onChange={(event) => setName(event.currentTarget.value)}
          />
        </TextField>
        <TextField className="mt-4" fullWidth isRequired>
          <Label>Schedule</Label>
          <Input
            className="font-mono"
            value={schedule}
            maxLength={100}
            onChange={(event) => setSchedule(event.currentTarget.value)}
          />
        </TextField>
        <TextField className="mt-4" fullWidth isRequired>
          <Label>Command</Label>
          <Input
            className="font-mono"
            value={command}
            maxLength={1_000}
            onChange={(event) => setCommand(event.currentTarget.value)}
          />
        </TextField>
        <Button
          className="mt-5"
          type="submit"
          variant="primary"
          fullWidth
          isDisabled={!selectedAccount || pending !== ""}
        >
          <Plus className="size-4" />
          Add cron job
        </Button>
      </form>
    </div>
  );
};
