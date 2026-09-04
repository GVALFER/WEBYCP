import { Button, Input, Label, TextField, toast } from "@heroui/react";
import { ArchiveRestore, HardDrive, Play, Plus, Trash2 } from "lucide-react";
import { type FormEvent, useState } from "react";
import useSWR, { useSWRConfig } from "swr";
import * as v from "valibot";

import type {
  AccountListResponse,
  BackupArtifactListResponse,
  BackupManifest,
  BackupPlan,
  BackupPlanListResponse,
  BackupRunListResponse,
  BackupRunResponse,
  Job,
} from "../../api/types";
import { Confirm } from "../../components/Confirm";
import { SelectField } from "../../components/SelectField";
import { api, fetcher } from "../../lib/api";
import { cn } from "../../utils/classnames";
import { formatDate } from "../../utils/date";
import { errorMessage } from "../../utils/errors";
import { statusClass } from "../../utils/status";
import {
  issueMessage,
  nameField,
  scheduleField,
} from "../../utils/validation";

const planSchema = v.object({
  name: nameField,
  schedule: scheduleField,
  retentionCount: v.pipe(
    v.string(),
    v.regex(/^\d+$/, "Enter a whole retention number."),
    v.transform(Number),
    v.integer("Enter a whole retention number."),
    v.minValue(1, "Keep at least one backup."),
    v.maxValue(100, "Keep no more than 100 backups."),
  ),
});

export const BackupsPanel = () => {
  const [accountId, setAccountId] = useState("");
  const [name, setName] = useState("Daily backup");
  const [schedule, setSchedule] = useState("0 3 * * *");
  const [retention, setRetention] = useState("7");
  const [files, setFiles] = useState(true);
  const [databases, setDatabases] = useState(true);
  const [pending, setPending] = useState("");
  const { mutate: mutateKey } = useSWRConfig();
  const { data: accounts } = useSWR<AccountListResponse>("accounts", fetcher);
  const { data: plans, mutate: mutatePlans } =
    useSWR<BackupPlanListResponse>("backup-plans", fetcher);
  const { data: runs, mutate: mutateRuns } =
    useSWR<BackupRunListResponse>("backup-runs", fetcher);
  const { data: artifacts, mutate: mutateArtifacts } =
    useSWR<BackupArtifactListResponse>("backup-artifacts", fetcher);
  const activeAccounts =
    accounts?.items.filter(
      (account) => account.status === "active" && account.enabled,
    ) ?? [];
  const selectedAccount = accountId || activeAccounts[0]?.id || "";

  const refresh = () =>
    Promise.all([
      mutatePlans(),
      mutateRuns(),
      mutateArtifacts(),
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
      toast.danger("Backup action failed", {
        description: await errorMessage(error),
      });
    } finally {
      setPending("");
    }
  };

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!files && !databases) {
      toast.warning("Choose what to back up");
      return;
    }
    const result = v.safeParse(planSchema, {
      name,
      schedule,
      retentionCount: retention,
    });
    if (!result.success) {
      toast.warning("Check the backup plan", {
        description: issueMessage(result.issues),
      });
      return;
    }
    await run(
      "create",
      () =>
        api
          .post("backup-plans", {
            json: {
              accountId: selectedAccount,
              ...result.output,
              includeFiles: files,
              includeDatabases: databases,
              enabled: true,
            },
          })
          .json<BackupPlan>(),
      "Backup plan created",
    );
  };

  const restore = (id: string) =>
    run(
      id,
      async () => {
        const manifest = await api
          .get(`backup-artifacts/${encodeURIComponent(id)}/restore`)
          .json<BackupManifest>();
        await api
          .post(`backup-artifacts/${encodeURIComponent(id)}/restore`, {
            json: {
              files: manifest.files,
              databases: manifest.databases,
              metadata: manifest.metadata,
            },
          })
          .json<Job>();
      },
      "Restore queued",
    );

  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_22rem]">
      <div className="space-y-6">
        <section className="panel-card overflow-hidden">
          <div className="border-b border-divider px-6 py-5">
            <h2 className="text-base font-semibold">Backup plans</h2>
            <div className="mt-1 text-sm text-foreground-500">
              Scheduled and on-demand local backups with retention.
            </div>
          </div>
          <div className="divide-y divide-divider">
            {plans?.items.length ? (
              plans.items.map((plan) => (
                <div
                  key={plan.id}
                  className="flex flex-col gap-4 px-6 py-5 sm:flex-row sm:items-center sm:justify-between"
                >
                  <div className="flex items-center gap-4">
                    <div className="icon-box">
                      <HardDrive className="size-5" />
                    </div>
                    <div>
                      <div className="font-medium">{plan.name}</div>
                      <div className="mt-1 text-xs text-foreground-400">
                        {plan.schedule || "Manual only"} · keep {plan.retentionCount}
                        {plan.nextRunAt
                          ? ` · next ${formatDate(plan.nextRunAt)}`
                          : ""}
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      size="sm"
                      variant="secondary"
                      isDisabled={pending !== ""}
                      onPress={() =>
                        void run(
                          plan.id,
                          () =>
                            api
                              .post(
                                `backup-plans/${encodeURIComponent(plan.id)}/runs`,
                              )
                              .json<BackupRunResponse>(),
                          "Backup queued",
                        )
                      }
                    >
                      <Play className="size-4" />
                      Run now
                    </Button>
                    <Confirm
                      title={`Delete ${plan.name}?`}
                      description="The schedule will be removed. Existing backup artifacts will be retained."
                      action="Delete plan"
                      onConfirm={() =>
                        void run(
                          plan.id,
                          async () => {
                            await api.delete(
                              `backup-plans/${encodeURIComponent(plan.id)}`,
                            );
                          },
                          "Backup plan deleted",
                        )
                      }
                    >
                      <Button
                        isIconOnly
                        size="sm"
                        variant="danger-soft"
                        aria-label={`Delete ${plan.name}`}
                        isDisabled={pending !== ""}
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </Confirm>
                  </div>
                </div>
              ))
            ) : (
              <div className="px-6 py-12 text-center text-sm text-foreground-400">
                No backup plans yet.
              </div>
            )}
          </div>
        </section>

        <section className="panel-card overflow-hidden">
          <div className="border-b border-divider px-6 py-5">
            <h2 className="text-base font-semibold">Verified artifacts</h2>
          </div>
          <div className="divide-y divide-divider">
            {artifacts?.items.length ? (
              artifacts.items.map((artifact) => (
                <div
                  key={artifact.id}
                  className="flex flex-col gap-4 px-6 py-4 sm:flex-row sm:items-center sm:justify-between"
                >
                  <div>
                    <div className="font-medium">
                      {formatDate(artifact.createdAt)}
                    </div>
                    <div className="mt-1 text-xs text-foreground-400">
                      {(artifact.size / 1_048_576).toFixed(2)} MB · SHA-256{" "}
                      {artifact.checksum.slice(0, 12)}…
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Confirm
                      title="Restore this backup?"
                      description="The verified manifest will be restored. Existing files and databases may be overwritten."
                      action="Restore backup"
                      onConfirm={() => void restore(artifact.id)}
                    >
                      <Button
                        size="sm"
                        variant="secondary"
                        isDisabled={pending !== ""}
                      >
                        <ArchiveRestore className="size-4" />
                        Restore
                      </Button>
                    </Confirm>
                    <Confirm
                      title="Delete this backup?"
                      description="The local backup artifact will be permanently deleted."
                      action="Delete backup"
                      onConfirm={() =>
                        void run(
                          artifact.id,
                          async () => {
                            await api.delete(
                              `backup-artifacts/${encodeURIComponent(artifact.id)}`,
                            );
                          },
                          "Backup artifact deleted",
                        )
                      }
                    >
                      <Button
                        isIconOnly
                        size="sm"
                        variant="danger-soft"
                        aria-label="Delete backup artifact"
                        isDisabled={pending !== ""}
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </Confirm>
                  </div>
                </div>
              ))
            ) : (
              <div className="px-6 py-10 text-center text-sm text-foreground-400">
                No completed artifacts yet.
              </div>
            )}
          </div>
        </section>

        <section className="panel-card p-6">
          <h2 className="text-base font-semibold">Recent runs</h2>
          <div className="mt-4 space-y-2">
            {runs?.items.length ? (
              runs.items.slice(0, 10).map((item) => (
                <div
                  key={item.id}
                  className="flex items-center justify-between rounded-xl border border-border/70 bg-surface-secondary/60 px-4 py-3 text-sm"
                >
                  <div>{formatDate(item.createdAt)}</div>
                  <span
                    className={cn(
                      "rounded-full px-2 py-1 text-xs capitalize",
                      statusClass(item.status),
                    )}
                  >
                    {item.status}
                  </span>
                </div>
              ))
            ) : (
              <div className="py-6 text-center text-sm text-foreground-400">
                No backup runs yet.
              </div>
            )}
          </div>
        </section>
      </div>

      <form className="panel-card h-fit p-6" onSubmit={submit}>
        <h2 className="text-base font-semibold">Create backup plan</h2>
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
        <TextField className="mt-4" fullWidth>
          <Label>Schedule (blank for manual)</Label>
          <Input
            className="font-mono"
            value={schedule}
            maxLength={100}
            onChange={(event) => setSchedule(event.currentTarget.value)}
          />
        </TextField>
        <TextField className="mt-4" fullWidth isRequired>
          <Label>Retention</Label>
          <Input
            type="number"
            min="1"
            max="100"
            value={retention}
            onChange={(event) => setRetention(event.currentTarget.value)}
          />
        </TextField>
        <label className="mt-5 flex items-center gap-3 text-sm">
          <input
            type="checkbox"
            checked={files}
            onChange={(event) => setFiles(event.currentTarget.checked)}
          />
          Files
        </label>
        <label className="mt-3 flex items-center gap-3 text-sm">
          <input
            type="checkbox"
            checked={databases}
            onChange={(event) => setDatabases(event.currentTarget.checked)}
          />
          Databases
        </label>
        <Button
          className="mt-6"
          type="submit"
          variant="primary"
          fullWidth
          isDisabled={
            !selectedAccount || (!files && !databases) || pending !== ""
          }
        >
          <Plus className="size-4" />
          Create plan
        </Button>
      </form>
    </div>
  );
};
