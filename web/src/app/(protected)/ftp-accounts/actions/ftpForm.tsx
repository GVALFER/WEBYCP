"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { toast } from "@heroui/react";
import { Pencil } from "lucide-react";
import { useCallback, useState, useTransition } from "react";
import { useForm, useWatch } from "react-hook-form";
import { useSWRConfig } from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormCheckbox } from "@/components/form/formCheckbox";
import { FormInput } from "@/components/form/formInput";
import { FormModal } from "@/components/form/formModal";
import { FormSelect } from "@/components/form/formSelect";
import type { AccountListResponse, FTPAccount, FTPAccountResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";
import { passwordField, usernameField } from "@/utils/validation";

const ftpPassword = v.pipe(passwordField, v.minLength(12, "Use at least 12 characters."));
const formSchema = (editing: boolean) => v.object({
    accountId: v.pipe(v.string(), v.nonEmpty("Choose a hosting account.")),
    username: usernameField,
    password: editing ? v.union([v.literal(""), ftpPassword]) : ftpPassword,
    enabled: v.boolean(),
});

type Values = v.InferOutput<ReturnType<typeof formSchema>>;
type Props = {
    accounts?: AccountListResponse["items"];
    item?: FTPAccount;
};

const FTPForm = ({ accounts = [], item }: Props) => {
    const [open, setOpen] = useState(false);
    const [pending, startTransition] = useTransition();
    const { mutate } = useSWRConfig();
    const options = accounts.filter((account) => account.status === "active" && account.enabled);
    const defaults: Values = {
        accountId: item?.accountId ?? options[0]?.id ?? "",
        username: item?.username ?? "",
        password: "",
        enabled: item?.enabled ?? true,
    };
    const form = useForm<Values>({
        resolver: valibotResolver(formSchema(!!item)),
        defaultValues: defaults,
    });
    const accountId = useWatch({ control: form.control, name: "accountId" });
    const account = options.find((value) => value.id === accountId);

    const changeOpen = (value: boolean) => {
        if (pending) return;
        form.reset(defaults);
        setOpen(value);
    };

    const handleSubmit = useCallback((values: Values) => {
        startTransition(async () => {
            try {
                if (item) {
                    await api.patch(`ftp-accounts/${encodeURIComponent(item.id)}`, {
                        json: {
                            username: values.username,
                            enabled: values.enabled,
                            ...(values.password ? { password: values.password } : {}),
                        },
                    }).json<FTPAccountResponse>();
                } else {
                    await api.post("ftp-accounts", { json: values }).json<FTPAccountResponse>();
                }
                form.resetField("password");
                setOpen(false);
                await mutate((key) => isPageKey(key, "ftp-accounts", "accounts", "jobs", "audit-events"));
                toast.success(item ? "FTP update queued" : "FTP account creation queued");
            } catch (error) {
                toast.danger("FTP action failed", { description: await errorMessage(error) });
            }
        });
    }, [form, item, mutate]);

    return (
        <Form {...form}>
            <FormModal
                open={open}
                title={item ? "Edit FTP account" : "Add FTP account"}
                description="Changes disconnect all FTP sessions for this hosting account. Other accounts are unaffected."
                triggerLabel={item ? `Edit ${item.username}` : "Add FTP account"}
                triggerIcon={item ? <Pencil className="size-4" /> : undefined}
                triggerVariant={item ? "secondary" : "primary"}
                submitLabel={item ? "Save FTP account" : "Add FTP account"}
                pending={pending}
                submitDisabled={!accountId}
                onOpenChange={changeOpen}
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                {!item && (
                    <FormSelect
                        name="accountId"
                        label="Hosting account"
                        options={options}
                        empty="No active accounts"
                        required
                    />
                )}
                <FormInput name="username" label="Username" autoComplete="off" maxLength={32} required />
                <FormInput
                    name="password"
                    label={item ? "New password (leave blank to keep current)" : "Password"}
                    type="password"
                    autoComplete="new-password"
                    maxLength={128}
                    required={!item}
                />
                {(item || account) && (
                    <div className="rounded-xl bg-default px-4 py-3 text-xs text-foreground-500">
                        <div className="font-mono">{item?.home ?? `/home/${account?.systemUser}`}</div>
                        <div className="mt-1">Account home only. No SSH or SFTP access.</div>
                        {account && (
                            <div className="mt-2">
                                FTP accounts: {account.usage.ftpAccounts} / {account.package.limits.ftpAccounts}
                            </div>
                        )}
                    </div>
                )}
                <FormCheckbox name="enabled" label="Enable FTP login" />
            </FormModal>
        </Form>
    );
};

export default FTPForm;
