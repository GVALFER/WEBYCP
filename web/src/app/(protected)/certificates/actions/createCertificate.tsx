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
import type { CertificateJobResponse, DomainListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey, pageKey } from "@/utils/pagination";
import { emailField } from "@/utils/validation";

type CreateCertificateProps = {
    domains: DomainListResponse;
    email: string;
};

const formSchema = v.object({
    domainId: v.pipe(v.string(), v.nonEmpty("Choose a domain.")),
    email: emailField,
});

type FormValues = v.InferOutput<typeof formSchema>;

const CreateCertificate = ({ domains, email }: CreateCertificateProps) => {
    const [open, setOpen] = useState(false);

    const [pending, startTransition] = useTransition();
    const { mutate } = useSWRConfig();

    const { data } = useSWR<DomainListResponse>(pageKey("domains", { page: 1, size: 100 }), {
        fallbackData: domains,
    });

    const options =
        data?.items.filter((domain) => domain.status === "active" && domain.enabled) ?? [];

    const form = useForm<FormValues>({
        resolver: valibotResolver(formSchema),
        defaultValues: { domainId: options[0]?.id ?? "", email },
    });

    const domainId = useWatch({ control: form.control, name: "domainId" });

    const handleSubmit = useCallback(
        (values: FormValues) => {
            startTransition(async () => {
                try {
                    await api
                        .post(`domains/${encodeURIComponent(values.domainId)}/certificate`, {
                            json: { email: values.email },
                        })
                        .json<CertificateJobResponse>();
                    await mutate((key) => isPageKey(key, "certificates", "jobs"));
                    setOpen(false);
                    toast.success("Certificate request queued");
                } catch (error) {
                    toast.danger("Certificate action failed", {
                        description: await errorMessage(error),
                    });
                }
            });
        },
        [mutate],
    );

    return (
        <Form {...form}>
            <FormModal
                open={open}
                title="Secure a domain"
                description="Requests a Let's Encrypt certificate for an active domain."
                triggerLabel="Secure a domain"
                triggerText="Domain"
                submitLabel="Issue certificate"
                pending={pending}
                submitDisabled={!domainId}
                onOpenChange={setOpen}
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                <FormSelect
                    name="domainId"
                    label="Domain"
                    options={options}
                    empty="No active domains"
                    required
                />
                <FormInput name="email" label="ACME email" type="email" required />
            </FormModal>
        </Form>
    );
};

export default CreateCertificate;
