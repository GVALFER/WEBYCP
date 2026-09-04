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
import type { WebsiteDomainJobResponse, WebsiteListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";
import { domainField } from "@/utils/validation";

const schema = v.object({
    websiteId: v.pipe(v.string(), v.nonEmpty("Choose a website.")),
    hostname: domainField,
});

type Values = v.InferOutput<typeof schema>;

const CreateAlias = ({ websites }: { websites: WebsiteListResponse }) => {
    const [open, setOpen] = useState(false);
    const [pending, startTransition] = useTransition();
    const { mutate } = useSWRConfig();

    const options = websites.items.filter((item) => item.status === "active" && item.enabled);

    const form = useForm<Values>({
        resolver: valibotResolver(schema),
        defaultValues: {
            websiteId: options[0]?.id ?? "",
            hostname: "",
        },
    });

    const handleSubmit = useCallback(
        (values: Values) => {
            startTransition(async () => {
                try {
                    await api
                        .post(`websites/${encodeURIComponent(values.websiteId)}/domains`, {
                            json: { hostname: values.hostname },
                        })
                        .json<WebsiteDomainJobResponse>();
                    form.reset({ websiteId: values.websiteId, hostname: "" });
                    await mutate((key) => isPageKey(key, "website-domains", "accounts", "jobs"));
                    setOpen(false);
                    toast.success("Alias queued for creation");
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
                title="Add alias"
                description="Adds another hostname to an active website."
                triggerLabel="Add alias"
                submitLabel="Add alias"
                pending={pending}
                submitDisabled={!options.length}
                onOpenChange={setOpen}
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                <FormSelect
                    name="websiteId"
                    label="Website"
                    options={options}
                    empty="No active websites"
                    required
                />
                <FormInput
                    name="hostname"
                    label="Hostname"
                    placeholder="www.example.com"
                    maxLength={253}
                    required
                />
            </FormModal>
        </Form>
    );
};

export default CreateAlias;
