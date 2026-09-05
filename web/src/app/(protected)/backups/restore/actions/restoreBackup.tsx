"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { Spinner, toast } from "@heroui/react";
import { ArchiveRestore } from "lucide-react";
import { useCallback, useState, useTransition } from "react";
import { useForm } from "react-hook-form";
import { useSWRConfig } from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormCheckbox } from "@/components/form/formCheckbox";
import { FormModal } from "@/components/form/formModal";
import type { BackupArtifact, BackupManifest, Job } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";

const formSchema = v.pipe(
    v.object({ files: v.boolean(), databases: v.boolean(), metadata: v.boolean() }),
    v.forward(
        v.check((scope) => scope.files || scope.databases || scope.metadata, "Select at least one scope."),
        ["files"],
    ),
);

type Values = v.InferOutput<typeof formSchema>;
type Props = { archive: BackupArtifact };

const RestoreBackup = ({ archive }: Props) => {
    const [open, setOpen] = useState(false);
    const [manifest, setManifest] = useState<BackupManifest>();
    const [verifying, startVerify] = useTransition();
    const [pending, startTransition] = useTransition();
    const { mutate } = useSWRConfig();
    const { dt } = useTimezone();
    const path = `backup-artifacts/${encodeURIComponent(archive.id)}/restore`;
    const form = useForm<Values>({
        resolver: valibotResolver(formSchema),
        defaultValues: { files: false, databases: false, metadata: false },
    });

    const changeOpen = (value: boolean) => {
        if (pending || verifying) return;
        setOpen(value);
        if (!value) return;
        setManifest(undefined);
        startVerify(async () => {
            try {
                const verified = await api.get(path).json<BackupManifest>();
                setManifest(verified);
                form.reset({ files: verified.files, databases: verified.databases, metadata: verified.metadata });
            } catch (error) {
                setOpen(false);
                toast.danger("Archive verification failed", { description: await errorMessage(error) });
            }
        });
    };

    const handleSubmit = useCallback((values: Values) => {
        startTransition(async () => {
            try {
                await api.post(path, { json: values }).json<Job>();
                await mutate((key) => isPageKey(key, "jobs"));
                setOpen(false);
                toast.success("Restore queued", { description: "Follow its progress in System → Jobs." });
            } catch (error) {
                toast.danger("Restore failed", { description: await errorMessage(error) });
            }
        });
    }, [mutate, path]);

    return (
        <Form {...form}>
            <FormModal
                open={open}
                title="Restore backup"
                description="Selected content will overwrite matching data in the original account. Take a fresh backup before continuing."
                triggerLabel={`Restore archive ${archive.id}`}
                triggerText="Restore"
                triggerIcon={<ArchiveRestore className="size-4" />}
                triggerVariant="secondary"
                submitLabel="Restore selected content"
                submitDisabled={!manifest || verifying}
                pending={pending}
                size="lg"
                onOpenChange={changeOpen}
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                {verifying ? (
                    <div className="flex justify-center py-8" aria-label="Verifying archive">
                        <Spinner size="sm" />
                    </div>
                ) : manifest && (
                    <>
                        <div className="rounded-xl border border-success/25 bg-success/8 p-4 text-sm">
                            <div className="font-medium text-success">Archive integrity verified</div>
                            <div className="mt-2 text-foreground-500">
                                {dt(manifest.createdAt)} · {manifest.entries.length} verified entries
                            </div>
                            <div className="mt-1 break-all font-mono text-xs text-foreground-400">
                                Account: {manifest.accountId}
                            </div>
                        </div>
                        <div className="space-y-3">
                            {manifest.files && <FormCheckbox name="files" label="Account files" />}
                            {manifest.databases && <FormCheckbox name="databases" label="Database contents" />}
                            {manifest.metadata && <FormCheckbox name="metadata" label="Website, database and scheduled task definitions" />}
                            {!manifest.files && <div className="text-sm text-danger" role="alert">{form.formState.errors.files?.message}</div>}
                        </div>
                        <div className="text-xs text-foreground-400">
                            Only content present in this archive is available. Database users, passwords,
                            grants, DNS zones and certificate keys are not included.
                            Restores overlay existing data; unrelated content is not removed.
                        </div>
                    </>
                )}
            </FormModal>
        </Form>
    );
};

export default RestoreBackup;
