"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { toast } from "@heroui/react";
import { useCallback, useState, useTransition } from "react";
import { useForm } from "react-hook-form";
import { useSWRConfig } from "swr";
import { Form } from "@/components/form/form";
import { FormModal } from "@/components/form/formModal";
import type { DNSRecordJobResponse, DNSZone } from "@/contracts/types";
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
    defaultTtl: number;
    zone: DNSZone;
};

const CreateRecord = ({ defaultTtl, zone }: Props) => {
    const [open, setOpen] = useState(false);
    const [pending, startTransition] = useTransition();
    const { mutate } = useSWRConfig();
    const path = `dns/zones/${encodeURIComponent(zone.id)}/records`;
    const form = useForm<RecordValues>({
        resolver: valibotResolver(recordSchema),
        defaultValues: recordValues(undefined, defaultTtl),
    });

    const handleSubmit = useCallback(
        (values: RecordValues) => {
            startTransition(async () => {
                try {
                    await api
                        .post(path, { json: requestValues(values) })
                        .json<DNSRecordJobResponse>();
                    form.reset(recordValues(undefined, defaultTtl));
                    await mutate((key) => isPageKey(key, path, "jobs"));
                    setOpen(false);
                    toast.success("DNS record queued for creation");
                } catch (error) {
                    toast.danger("DNS record could not be created", {
                        description: await errorMessage(error),
                    });
                }
            });
        },
        [defaultTtl, form, mutate, path],
    );

    return (
        <Form {...form}>
            <FormModal
                open={open}
                title="Create DNS record"
                description={`Add a record to ${zone.name}. Use @ for the zone apex.`}
                triggerLabel="Create DNS record"
                submitLabel="Create record"
                pending={pending}
                size="lg"
                submitDisabled={zone.status !== "active"}
                onOpenChange={setOpen}
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                <RecordFields />
            </FormModal>
        </Form>
    );
};

export default CreateRecord;
