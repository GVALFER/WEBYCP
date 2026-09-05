"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { Button, toast } from "@heroui/react";
import { Pencil, Trash2 } from "lucide-react";
import { useCallback, useState, useTransition } from "react";
import { useForm } from "react-hook-form";
import { useSWRConfig } from "swr";
import { Confirm } from "@/components/actions/confirm";
import { Form } from "@/components/form/form";
import { FormModal } from "@/components/form/formModal";
import type { DNSRecord, DNSRecordJobResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";
import RecordFields, {
    recordSchema,
    recordValues,
    requestValues,
    type RecordValues,
} from "./recordFields";

type Props = {
    record: DNSRecord;
    zoneName: string;
};

const RecordActions = ({ record, zoneName }: Props) => {
    const [open, setOpen] = useState(false);
    const [deleting, setDeleting] = useState(false);
    const [pending, startTransition] = useTransition();
    const { mutate } = useSWRConfig();
    const path = `dns/records/${encodeURIComponent(record.id)}`;
    const recordsPath = `dns/zones/${encodeURIComponent(record.zoneId)}/records`;
    const form = useForm<RecordValues>({
        resolver: valibotResolver(recordSchema),
        defaultValues: recordValues(record),
    });

    const refresh = useCallback(
        () => mutate((key) => isPageKey(key, recordsPath, "jobs")),
        [mutate, recordsPath],
    );

    const handleSubmit = useCallback(
        (values: RecordValues) => {
            startTransition(async () => {
                try {
                    await api
                        .patch(path, { json: requestValues(values) })
                        .json<DNSRecordJobResponse>();
                    await refresh();
                    setOpen(false);
                    toast.success("DNS record queued for update");
                } catch (error) {
                    toast.danger("DNS record could not be updated", {
                        description: await errorMessage(error),
                    });
                }
            });
        },
        [path, refresh],
    );

    const remove = async () => {
        setDeleting(true);
        try {
            await api.delete(path).json<DNSRecordJobResponse>();
            await refresh();
            toast.success("DNS record queued for deletion");
        } catch (error) {
            toast.danger("DNS record could not be deleted", {
                description: await errorMessage(error),
            });
        } finally {
            setDeleting(false);
        }
    };

    const disabled = record.status === "pending" || record.status === "deleting";

    return (
        <div className="flex items-center gap-1">
            <Form {...form}>
                <FormModal
                    open={open}
                    title={`Edit ${record.name}`}
                    description={`Update this record in ${zoneName}.`}
                    triggerLabel={`Edit ${record.name}`}
                    triggerIcon={<Pencil className="size-4" aria-hidden="true" />}
                    triggerVariant="secondary"
                    submitLabel="Save record"
                    pending={pending}
                    size="lg"
                    submitDisabled={disabled}
                    onOpenChange={(value) => {
                        setOpen(value);
                        if (value) form.reset(recordValues(record));
                    }}
                    onSubmit={form.handleSubmit(handleSubmit)}
                >
                    <RecordFields />
                </FormModal>
            </Form>
            <Confirm
                title={`Delete ${record.name} ${record.type}?`}
                description="This record will be removed from the authoritative zone."
                action="Delete record"
                onConfirm={remove}
            >
                <Button
                    isIconOnly
                    size="sm"
                    variant="danger-soft"
                    aria-label={`Delete ${record.name}`}
                    isDisabled={disabled || deleting}
                >
                    <Trash2 className="size-4" aria-hidden="true" />
                </Button>
            </Confirm>
        </div>
    );
};

export default RecordActions;
