"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { toast } from "@heroui/react";
import { useCallback, useState, useTransition } from "react";
import { useForm, useWatch } from "react-hook-form";
import useSWR, { useSWRConfig } from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormInput } from "@/components/form/formInput";
import { FormModal } from "@/components/form/formModal";
import { FormSelect } from "@/components/form/formSelect";
import type { AccountListResponse, DomainJobResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { domainField } from "@/utils/validation";

type CreateDomainProps = {
    accounts: AccountListResponse;
};

const formSchema = v.object({
    accountId: v.pipe(v.string(), v.nonEmpty("Choose a hosting account.")),
    name: domainField,
});

type FormValues = v.InferOutput<typeof formSchema>;

const CreateDomain = ({ accounts }: CreateDomainProps) => {
    const [open, setOpen] = useState(false);
    const [pending, startTransition] = useTransition();
    const { mutate } = useSWRConfig();

    const { data } = useSWR<AccountListResponse>("accounts", {
        fallbackData: accounts,
    });

    const options = data?.items.filter((account) => account.status === "active") ?? [];

    const form = useForm<FormValues>({
        resolver: valibotResolver(formSchema),
        defaultValues: { accountId: options[0]?.id ?? "", name: "" },
    });

    const accountId = useWatch({ control: form.control, name: "accountId" });

    const handleSubmit = useCallback(
        (values: FormValues) => {
            startTransition(async () => {
                try {
                    await api.post("domains", { json: values }).json<DomainJobResponse>();
                    form.reset({ accountId: values.accountId, name: "" });
                    await Promise.all([mutate("domains"), mutate("jobs")]);
                    setOpen(false);
                    toast.success("Domain queued for creation");
                } catch (error) {
                    toast.danger("Domain action failed", {
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
                title="Add domain"
                description="Creates the document root and validates services before reloading."
                triggerLabel="Add domain"
                submitLabel="Add domain"
                pending={pending}
                submitDisabled={!accountId}
                onOpenChange={setOpen}
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                <FormSelect
                    name="accountId"
                    label="Hosting account"
                    options={options}
                    empty="No active accounts"
                    required
                />
                <FormInput
                    name="name"
                    label="Domain name"
                    placeholder="example.com"
                    autoCapitalize="none"
                    autoCorrect="off"
                    maxLength={253}
                    required
                />
            </FormModal>
        </Form>
    );
};

export default CreateDomain;
