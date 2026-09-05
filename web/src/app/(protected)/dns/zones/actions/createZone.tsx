"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { toast } from "@heroui/react";
import { useCallback, useState, useTransition } from "react";
import { useForm } from "react-hook-form";
import { useSWRConfig } from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormInput } from "@/components/form/formInput";
import { FormModal } from "@/components/form/formModal";
import { FormSelect } from "@/components/form/formSelect";
import type { AccountOverview, DNSProvider, DNSZoneJobResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";
import { domainField } from "@/utils/validation";

const schema = v.object({
    accountId: v.pipe(v.string(), v.nonEmpty("Choose a hosting account.")),
    providerId: v.pipe(v.string(), v.nonEmpty("Choose a DNS provider.")),
    name: domainField,
});

type Values = v.InferOutput<typeof schema>;

type Props = {
    accounts: AccountOverview[];
    configured: boolean;
    providers: DNSProvider[];
};

const CreateZone = ({ accounts, configured, providers }: Props) => {
    const [open, setOpen] = useState(false);
    const [pending, startTransition] = useTransition();
    const { mutate } = useSWRConfig();

    const activeAccounts = accounts.filter((item) => item.status === "active" && item.enabled);

    const form = useForm<Values>({
        resolver: valibotResolver(schema),
        defaultValues: {
            accountId: activeAccounts[0]?.id ?? "",
            providerId: providers[0]?.id ?? "",
            name: "",
        },
    });

    const handleSubmit = useCallback(
        (values: Values) => {
            startTransition(async () => {
                try {
                    await api.post("dns/zones", { json: values }).json<DNSZoneJobResponse>();
                    form.reset({ ...values, name: "" });
                    await mutate((key) => isPageKey(key, "dns/zones", "jobs"));
                    setOpen(false);
                    toast.success("DNS zone queued for creation");
                } catch (error) {
                    toast.danger("DNS zone could not be created", {
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
                title="Create DNS zone"
                description="Creates an authoritative zone on the selected account server."
                triggerLabel="Create DNS zone"
                submitLabel="Create zone"
                pending={pending}
                submitDisabled={!configured || !activeAccounts.length || !providers.length}
                onOpenChange={setOpen}
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                <FormSelect
                    name="accountId"
                    label="Hosting account"
                    options={activeAccounts}
                    empty="No active accounts"
                    required
                />
                <FormSelect
                    name="providerId"
                    label="DNS provider"
                    options={providers}
                    empty="No DNS providers"
                    required
                />
                <FormInput name="name" label="Zone name" maxLength={253} required />
            </FormModal>
        </Form>
    );
};

export default CreateZone;
