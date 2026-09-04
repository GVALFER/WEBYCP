"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { toast } from "@heroui/react";
import { useCallback, useEffect, useState, useTransition } from "react";
import { useForm, useWatch } from "react-hook-form";
import useSWR, { useSWRConfig } from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormInput } from "@/components/form/formInput";
import { FormModal } from "@/components/form/formModal";
import { FormSelect } from "@/components/form/formSelect";
import type { DomainAliasJobResponse, DomainListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey, pageKey } from "@/utils/pagination";
import { domainField } from "@/utils/validation";

type CreateAliasProps = {
    domains: DomainListResponse;
    domainId: string;
    onDomainChange: (id: string) => void;
};

const formSchema = v.object({
    domainId: v.pipe(v.string(), v.nonEmpty("Choose a primary domain.")),
    name: domainField,
});

type FormValues = v.InferOutput<typeof formSchema>;

const CreateAlias = ({ domains, domainId, onDomainChange }: CreateAliasProps) => {
    const [open, setOpen] = useState(false);
    const [pending, startTransition] = useTransition();
    const { mutate } = useSWRConfig();

    const { data } = useSWR<DomainListResponse>(pageKey("domains", { page: 1, size: 100 }), {
        fallbackData: domains,
    });

    const options = data?.items.filter((domain) => domain.status === "active") ?? [];

    const form = useForm<FormValues>({
        resolver: valibotResolver(formSchema),
        defaultValues: { domainId, name: "" },
    });

    const selected = useWatch({ control: form.control, name: "domainId" });

    useEffect(() => {
        form.setValue("domainId", domainId);
    }, [domainId, form]);

    const handleSubmit = useCallback(
        (values: FormValues) => {
            startTransition(async () => {
                try {
                    await api
                        .post(`domains/${encodeURIComponent(values.domainId)}/aliases`, {
                            json: { name: values.name },
                        })
                        .json<DomainAliasJobResponse>();
                    form.reset({ domainId: values.domainId, name: "" });
                    const aliases = `domains/${encodeURIComponent(values.domainId)}/aliases`;
                    await mutate((key) => isPageKey(key, aliases, "jobs"));
                    setOpen(false);
                    toast.success("Alias queued for creation");
                } catch (error) {
                    toast.danger("Alias action failed", {
                        description: await errorMessage(error),
                    });
                }
            });
        },
        [form, mutate],
    );

    return (
        <Form {...form}>
            <FormModal
                open={open}
                title="Add alias"
                description="Points another hostname at an existing domain."
                triggerLabel="Add alias"
                submitLabel="Add alias"
                pending={pending}
                submitDisabled={!selected}
                onOpenChange={setOpen}
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                <FormSelect
                    name="domainId"
                    label="Primary domain"
                    options={options}
                    empty="No active domains"
                    onValueChange={onDomainChange}
                    required
                />
                <FormInput
                    name="name"
                    label="Alias name"
                    placeholder="www.example.com"
                    autoCapitalize="none"
                    autoCorrect="off"
                    maxLength={253}
                    required
                />
            </FormModal>
        </Form>
    );
};

export default CreateAlias;
