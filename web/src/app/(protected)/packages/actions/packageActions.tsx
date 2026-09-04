"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { Button, toast } from "@heroui/react";
import { Pencil, Trash2 } from "lucide-react";
import { useCallback, useState, useTransition } from "react";
import { useForm } from "react-hook-form";
import { useSWRConfig } from "swr";
import { Confirm } from "@/components/actions/confirm";
import { Form } from "@/components/form/form";
import { FormModal } from "@/components/form/formModal";
import type { Package } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";
import { PackageFields, packageSchema, packageValues, type PackageValues } from "./packageFields";

type Props = {
    value: Package;
};

const PackageActions = ({ value }: Props) => {
    const { mutate } = useSWRConfig();
    const [open, setOpen] = useState(false);
    const [pending, startTransition] = useTransition();

    const form = useForm<PackageValues>({
        resolver: valibotResolver(packageSchema),
        defaultValues: packageValues(value),
    });

    const handleSubmit = useCallback(
        (values: PackageValues) => {
            startTransition(async () => {
                try {
                    await api
                        .patch(`packages/${encodeURIComponent(value.id)}`, { json: values })
                        .json<Package>();
                    await mutate((key) => isPageKey(key, "packages", "accounts"));
                    setOpen(false);
                    toast.success("Package updated");
                } catch (error) {
                    toast.danger("Package update failed", {
                        description: await errorMessage(error),
                    });
                }
            });
        },
        [mutate, value.id],
    );

    const remove = async () => {
        try {
            await api.delete(`packages/${encodeURIComponent(value.id)}`);
            await mutate((key) => isPageKey(key, "packages"));
            toast.success("Package deleted");
        } catch (error) {
            toast.danger("Package deletion failed", {
                description: await errorMessage(error),
            });
        }
    };

    return (
        <div className="flex items-center gap-1">
            <Form {...form}>
                <FormModal
                    open={open}
                    title={`Edit ${value.name}`}
                    description="Lower limits do not delete or disable existing resources."
                    triggerLabel={`Edit ${value.name}`}
                    triggerIcon={<Pencil className="size-4" aria-hidden="true" />}
                    triggerVariant="secondary"
                    submitLabel="Save changes"
                    pending={pending}
                    size="lg"
                    onOpenChange={(next) => {
                        setOpen(next);
                        if (next) form.reset(packageValues(value));
                    }}
                    onSubmit={form.handleSubmit(handleSubmit)}
                >
                    <PackageFields />
                </FormModal>
            </Form>
            <Confirm
                title={`Delete ${value.name}?`}
                description="Only Packages without assigned Accounts can be deleted."
                action="Delete Package"
                onConfirm={remove}
            >
                <Button
                    isIconOnly
                    size="sm"
                    variant="danger-soft"
                    aria-label={`Delete ${value.name}`}
                    isDisabled={value.accountCount > 0}
                >
                    <Trash2 className="size-4" aria-hidden="true" />
                </Button>
            </Confirm>
        </div>
    );
};

export default PackageActions;
