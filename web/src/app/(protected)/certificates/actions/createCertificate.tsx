"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { Button, toast } from "@heroui/react";
import { useCallback, useTransition } from "react";
import { useForm, useWatch } from "react-hook-form";
import useSWR, { useSWRConfig } from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormInput } from "@/components/form/formInput";
import { FormSelect } from "@/components/form/formSelect";
import type { CertificateJobResponse, DomainListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
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
    const { mutate } = useSWRConfig();
    const { data } = useSWR<DomainListResponse>("domains", {
        fallbackData: domains,
    });
    const options =
        data?.items.filter((domain) => domain.status === "active" && domain.enabled) ?? [];
    const [pending, startTransition] = useTransition();
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
                    await Promise.all([mutate("certificates"), mutate("jobs")]);
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
            <form className="panel-card p-6" onSubmit={form.handleSubmit(handleSubmit)}>
                <h2 className="text-base font-semibold">Secure a domain</h2>
                <FormSelect
                    className="mt-5"
                    name="domainId"
                    label="Domain"
                    options={options}
                    required
                />
                <FormInput
                    className="mt-4"
                    name="email"
                    label="ACME email"
                    type="email"
                    required
                />
                <Button
                    className="mt-5"
                    type="submit"
                    variant="primary"
                    fullWidth
                    isDisabled={!domainId || pending}
                >
                    Issue certificate
                </Button>
            </form>
        </Form>
    );
};

export default CreateCertificate;
