"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { Button } from "@heroui/react";
import { Plus } from "lucide-react";
import { useCallback, useTransition } from "react";
import { useForm } from "react-hook-form";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormInput } from "@/components/form/formInput";
import type { DatabaseJobResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { dbNameField } from "@/utils/validation";
import { useDatabaseAction } from "./useDatabaseAction";

type CreateDatabaseProps = {
    accountId: string;
};

const formSchema = v.object({ name: dbNameField });
type FormValues = v.InferOutput<typeof formSchema>;

const CreateDatabase = ({ accountId }: CreateDatabaseProps) => {
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
                            .post("databases", { json: { accountId, ...values } })
                            .json<DatabaseJobResponse>(),
                    "Database queued for creation",
                );
                if (response) form.reset();
            });
        },
        [accountId, form, run],
    );

    return (
        <Form {...form}>
            <form className="mt-5 space-y-4" onSubmit={form.handleSubmit(handleSubmit)}>
                <FormInput name="name" label="Database name" maxLength={32} required />
                <Button
                    type="submit"
                    variant="primary"
                    fullWidth
                    isDisabled={!accountId || pending}
                >
                    <Plus className="size-4" />
                    Create database
                </Button>
            </form>
        </Form>
    );
};

export default CreateDatabase;
