"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { toast } from "@heroui/react";
import { PackageOpen } from "lucide-react";
import { useCallback, useState, useTransition } from "react";
import { useForm } from "react-hook-form";
import { useSWRConfig } from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormModal } from "@/components/form/formModal";
import { FormSelect } from "@/components/form/formSelect";
import type { Account, AccountOverview, Package } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";

type Props = {
    account: Account;
    packageId: string;
    packages: Package[];
};

const schema = v.object({
    packageId: v.pipe(v.string(), v.nonEmpty("Select a Package.")),
});
type Values = v.InferOutput<typeof schema>;

const AssignPackage = ({ account, packageId, packages }: Props) => {
    const { mutate } = useSWRConfig();
    const [open, setOpen] = useState(false);
    const [pending, startTransition] = useTransition();

    const form = useForm<Values>({
        resolver: valibotResolver(schema),
        defaultValues: { packageId },
    });

    const handleSubmit = useCallback(
        (values: Values) => {
            startTransition(async () => {
                try {
                    await api
                        .put(`accounts/${encodeURIComponent(account.id)}/package`, {
                            json: values,
                        })
                        .json<AccountOverview>();
                    await mutate((key) => isPageKey(key, "accounts", "packages"));
                    setOpen(false);
                    toast.success("Account Package updated");
                } catch (error) {
                    toast.danger("Package assignment failed", {
                        description: await errorMessage(error),
                    });
                }
            });
        },
        [account.id, mutate],
    );

    return (
        <Form {...form}>
            <FormModal
                open={open}
                title={`Assign Package to ${account.name}`}
                description="New limits apply immediately without deleting existing resources."
                triggerLabel={`Assign Package to ${account.name}`}
                triggerIcon={<PackageOpen className="size-4" aria-hidden="true" />}
                triggerVariant="secondary"
                submitLabel="Assign Package"
                pending={pending}
                submitDisabled={!packages.length}
                onOpenChange={(value) => {
                    setOpen(value);
                    if (value) form.reset({ packageId });
                }}
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                <FormSelect
                    name="packageId"
                    label="Package"
                    options={packages.map((item) => ({ id: item.id, name: item.name }))}
                    required
                />
            </FormModal>
        </Form>
    );
};

export default AssignPackage;
