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
import type { AccountJobResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";
import { nameField } from "@/utils/validation";

type CreateAccountProps = {
    nodeId: string;
};

const formSchema = v.object({ name: nameField });
type FormValues = v.InferOutput<typeof formSchema>;

const CreateAccount = ({ nodeId }: CreateAccountProps) => {
    const { mutate } = useSWRConfig();
    const [open, setOpen] = useState(false);
    const [pending, startTransition] = useTransition();

    const form = useForm<FormValues>({
        resolver: valibotResolver(formSchema),
        defaultValues: { name: "" },
    });

    const handleSubmit = useCallback(
        (values: FormValues) => {
            startTransition(async () => {
                try {
                    await api
                        .post("accounts", { json: { ...values, nodeId } })
                        .json<AccountJobResponse>();
                    form.reset();
                    await mutate((key) => isPageKey(key, "accounts", "jobs"));
                    setOpen(false);
                    toast.success("Account queued for creation");
                } catch (error) {
                    toast.danger("Account action failed", {
                        description: await errorMessage(error),
                    });
                }
            });
        },
        [form, mutate, nodeId],
    );

    return (
        <Form {...form}>
            <FormModal
                open={open}
                title="Create account"
                description="Queues an isolated Linux user on the selected node."
                triggerLabel="Create account"
                submitLabel="Create account"
                pending={pending}
                submitDisabled={!nodeId}
                onOpenChange={setOpen}
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                <FormInput name="name" label="Account name" maxLength={80} required />
            </FormModal>
        </Form>
    );
};

export default CreateAccount;
