"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { Button, toast } from "@heroui/react";
import { Plus } from "lucide-react";
import { useCallback, useTransition } from "react";
import { useForm } from "react-hook-form";
import { useSWRConfig } from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormInput } from "@/components/form/formInput";
import type { AccountJobResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { nameField } from "@/utils/validation";

type CreateAccountProps = {
    nodeId: string;
};

const formSchema = v.object({ name: nameField });
type FormValues = v.InferOutput<typeof formSchema>;

const CreateAccount = ({ nodeId }: CreateAccountProps) => {
    const { mutate } = useSWRConfig();
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
                    await Promise.all([mutate("accounts"), mutate("jobs")]);
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
            <form className="mt-6 space-y-5" onSubmit={form.handleSubmit(handleSubmit)}>
                <FormInput name="name" label="Account name" maxLength={80} required />
                <Button type="submit" variant="primary" fullWidth isDisabled={pending || !nodeId}>
                    <Plus className="size-4" aria-hidden="true" />
                    {pending ? "Queuing…" : "Create account"}
                </Button>
            </form>
        </Form>
    );
};

export default CreateAccount;
