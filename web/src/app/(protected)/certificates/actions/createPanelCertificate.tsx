"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { Button, toast } from "@heroui/react";
import { useCallback, useTransition } from "react";
import { useForm } from "react-hook-form";
import { useSWRConfig } from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormInput } from "@/components/form/formInput";
import type { CertificateJobResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { domainField, emailField } from "@/utils/validation";

type CreatePanelCertificateProps = {
    email: string;
};

const formSchema = v.object({
    hostname: domainField,
    email: emailField,
});

type FormValues = v.InferOutput<typeof formSchema>;

const CreatePanelCertificate = ({ email }: CreatePanelCertificateProps) => {
    const { mutate } = useSWRConfig();
    const [pending, startTransition] = useTransition();
    const form = useForm<FormValues>({
        resolver: valibotResolver(formSchema),
        defaultValues: { hostname: "", email },
    });

    const handleSubmit = useCallback(
        (values: FormValues) => {
            startTransition(async () => {
                try {
                    await api
                        .post("certificates/panel", { json: values })
                        .json<CertificateJobResponse>();
                    form.reset({ hostname: "", email: values.email });
                    await Promise.all([mutate("certificates"), mutate("jobs")]);
                    toast.success("Panel certificate request queued");
                } catch (error) {
                    toast.danger("Certificate action failed", {
                        description: await errorMessage(error),
                    });
                }
            });
        },
        [form, mutate],
    );

    return (
        <Form {...form}>
            <form className="panel-card p-6" onSubmit={form.handleSubmit(handleSubmit)}>
                <h2 className="text-base font-semibold">Panel certificate</h2>
                <div className="mt-1 text-sm text-foreground-500">
                    Use after the hostname resolves to this server.
                </div>
                <FormInput
                    className="mt-5"
                    name="hostname"
                    label="Panel hostname"
                    placeholder="panel.example.com"
                    maxLength={253}
                    required
                />
                <Button
                    className="mt-5"
                    type="submit"
                    variant="secondary"
                    fullWidth
                    isDisabled={pending}
                >
                    Secure panel
                </Button>
            </form>
        </Form>
    );
};

export default CreatePanelCertificate;
