"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { Button, toast } from "@heroui/react";
import { Plus } from "lucide-react";
import { useCallback, useEffect, useTransition } from "react";
import { useForm, useWatch } from "react-hook-form";
import useSWR, { useSWRConfig } from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormInput } from "@/components/form/formInput";
import { FormSelect } from "@/components/form/formSelect";
import type { DomainAliasJobResponse, DomainListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
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
    const { mutate } = useSWRConfig();
    const { data } = useSWR<DomainListResponse>("domains", {
        fallbackData: domains,
    });
    const options = data?.items.filter((domain) => domain.status === "active") ?? [];
    const [pending, startTransition] = useTransition();
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
                    await Promise.all([
                        mutate(`domains/${encodeURIComponent(values.domainId)}/aliases`),
                        mutate("jobs"),
                    ]);
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
        <aside className="panel-card p-6">
            <h2 className="text-base font-semibold">Add alias</h2>
            <div className="mt-1 text-sm leading-6 text-foreground-500">
                Points another hostname at an existing domain.
            </div>
            <Form {...form}>
                <form className="mt-6 space-y-5" onSubmit={form.handleSubmit(handleSubmit)}>
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
                    <Button
                        type="submit"
                        variant="primary"
                        fullWidth
                        isDisabled={pending || !selected}
                    >
                        <Plus className="size-4" aria-hidden="true" />
                        {pending ? "Queuing…" : "Add alias"}
                    </Button>
                </form>
            </Form>
        </aside>
    );
};

export default CreateAlias;
