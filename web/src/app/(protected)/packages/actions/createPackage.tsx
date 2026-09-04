"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { toast } from "@heroui/react";
import { useCallback, useState, useTransition } from "react";
import { useForm } from "react-hook-form";
import { useSWRConfig } from "swr";
import { Form } from "@/components/form/form";
import { FormModal } from "@/components/form/formModal";
import type { Package } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";
import { PackageFields, packageSchema, packageValues, type PackageValues } from "./packageFields";

const CreatePackage = () => {
    const { mutate } = useSWRConfig();
    const [open, setOpen] = useState(false);
    const [pending, startTransition] = useTransition();

    const form = useForm<PackageValues>({
        resolver: valibotResolver(packageSchema),
        defaultValues: packageValues(),
    });

    const handleSubmit = useCallback(
        (values: PackageValues) => {
            startTransition(async () => {
                try {
                    await api.post("packages", { json: values }).json<Package>();
                    form.reset(packageValues());
                    await mutate((key) => isPageKey(key, "packages"));
                    setOpen(false);
                    toast.success("Package created");
                } catch (error) {
                    toast.danger("Package creation failed", {
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
                title="Create Package"
                description="Set the resource ceilings for Accounts assigned to this Package."
                triggerLabel="Create Package"
                submitLabel="Create Package"
                pending={pending}
                size="lg"
                onOpenChange={setOpen}
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                <PackageFields />
            </FormModal>
        </Form>
    );
};

export default CreatePackage;
