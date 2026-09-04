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
import type { CertificateJobResponse, WebsiteListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey, pageKey } from "@/utils/pagination";
import { emailField } from "@/utils/validation";
import { useSession } from "@/providers/SessionProvider";

type CreateCertificateProps = {
    websites: WebsiteListResponse;
};

const formSchema = v.object({
    websiteId: v.pipe(v.string(), v.nonEmpty("Choose a website.")),
    email: emailField,
});

type FormValues = v.InferOutput<typeof formSchema>;

const CreateCertificate = ({ websites }: CreateCertificateProps) => {
    const [open, setOpen] = useState(false);
    const session = useSession();

    const [pending, startTransition] = useTransition();
    const { mutate } = useSWRConfig();

    const { data } = useSWR<WebsiteListResponse>(pageKey("websites", { page: 1, size: 100 }), {
        fallbackData: websites,
    });

    const options =
        data?.items.filter((website) => website.status === "active" && website.enabled) ?? [];

    const form = useForm<FormValues>({
        resolver: valibotResolver(formSchema),
        defaultValues: { websiteId: options[0]?.id ?? "", email: session.user.email },
    });

    const websiteId = useWatch({ control: form.control, name: "websiteId" });

    const handleSubmit = useCallback(
        (values: FormValues) => {
            startTransition(async () => {
                try {
                    await api
                        .post(`websites/${encodeURIComponent(values.websiteId)}/certificate`, {
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
                title="Secure a website"
                description="Requests a Let's Encrypt certificate for an active website and its domains."
                triggerLabel="Secure a website"
                triggerText="Website"
                submitLabel="Issue certificate"
                pending={pending}
                submitDisabled={!websiteId}
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
                <FormInput name="email" label="ACME email" type="email" required />
            </FormModal>
        </Form>
    );
};

export default CreateCertificate;
