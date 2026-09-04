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
import type { CertificateJobResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";
import { domainField, emailField } from "@/utils/validation";
import { useSession } from "@/providers/SessionProvider";

const formSchema = v.object({
    hostname: domainField,
    email: emailField,
});

type FormValues = v.InferOutput<typeof formSchema>;

const CreatePanelCertificate = () => {
    const [open, setOpen] = useState(false);
    const session = useSession();
    const { mutate } = useSWRConfig();
    const [pending, startTransition] = useTransition();

    const form = useForm<FormValues>({
        resolver: valibotResolver(formSchema),
        defaultValues: {
            hostname: "",
            email: session.user.email,
        },
    });

    const handleSubmit = useCallback(
        (values: FormValues) => {
            startTransition(async () => {
                try {
                    await api
                        .post("certificates/panel", { json: values })
                        .json<CertificateJobResponse>();
                    form.reset({ hostname: "", email: values.email });
                    await mutate((key) => isPageKey(key, "certificates", "jobs"));
                    setOpen(false);
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
            <FormModal
                open={open}
                title="Secure the panel"
                description="Use after the panel hostname resolves to this server."
                triggerLabel="Secure the panel"
                triggerText="Panel"
                submitLabel="Secure panel"
                pending={pending}
                onOpenChange={setOpen}
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                <FormInput
                    name="hostname"
                    label="Panel hostname"
                    placeholder="panel.example.com"
                    maxLength={253}
                    required
                />
            </FormModal>
        </Form>
    );
};

export default CreatePanelCertificate;
