"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { Button, toast } from "@heroui/react";
import { Plus } from "lucide-react";
import { useCallback, useTransition } from "react";
import { useForm, useWatch } from "react-hook-form";
import useSWR, { useSWRConfig } from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormInput } from "@/components/form/formInput";
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
    const { mutate } = useSWRConfig();
    const { data } = useSWR<AccountListResponse>("accounts", {
        fallbackData: accounts,
    });
    const options = data?.items.filter((account) => account.status === "active") ?? [];
    const [pending, startTransition] = useTransition();
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
        <aside className="panel-card p-6">
            <h2 className="text-base font-semibold">Add domain</h2>
            <div className="mt-1 text-sm leading-6 text-foreground-500">
                Creates the document root and validates services before reloading.
            </div>
            <Form {...form}>
                <form className="mt-6 space-y-5" onSubmit={form.handleSubmit(handleSubmit)}>
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
                    <Button
                        type="submit"
                        variant="primary"
                        fullWidth
                        isDisabled={pending || !accountId}
                    >
                        <Plus className="size-4" aria-hidden="true" />
                        {pending ? "Queuing…" : "Add domain"}
                    </Button>
                </form>
            </Form>
        </aside>
    );
};

export default CreateDomain;
