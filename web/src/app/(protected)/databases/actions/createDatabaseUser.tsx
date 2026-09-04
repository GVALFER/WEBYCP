"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { Button } from "@heroui/react";
import { KeyRound } from "lucide-react";
import { useCallback, useTransition } from "react";
import { useForm } from "react-hook-form";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormInput } from "@/components/form/formInput";
import type { DatabaseUserJobResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { dbNameField } from "@/utils/validation";
import { useDatabaseAction } from "./useDatabaseAction";

type CreateDatabaseUserProps = {
    accountId: string;
    onPassword: (password: string) => void;
};

const formSchema = v.object({ name: dbNameField });
type FormValues = v.InferOutput<typeof formSchema>;

const CreateDatabaseUser = ({ accountId, onPassword }: CreateDatabaseUserProps) => {
    const { run } = useDatabaseAction();
    const [pending, startTransition] = useTransition();
    const form = useForm<FormValues>({
        resolver: valibotResolver(formSchema),
        defaultValues: { name: "" },
    });

    const handleSubmit = useCallback(
        (values: FormValues) => {
            startTransition(async () => {
                const response = await run(
                    () =>
                        api
                            .post("database-users", { json: { accountId, ...values } })
                            .json<DatabaseUserJobResponse>(),
                    "Database user queued for creation",
                );
                if (response) {
                    form.reset();
                    onPassword(response.password ?? "");
                }
            });
        },
        [accountId, form, onPassword, run],
    );

    return (
        <Form {...form}>
            <form
                className="mt-6 space-y-4 border-t border-divider pt-6"
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                <FormInput name="name" label="User name" maxLength={32} required />
                <Button
                    type="submit"
                    variant="secondary"
                    fullWidth
                    isDisabled={!accountId || pending}
                >
                    <KeyRound className="size-4" />
                    Create user
                </Button>
            </form>
        </Form>
    );
};

export default CreateDatabaseUser;
