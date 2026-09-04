"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { Button } from "@heroui/react";
import { useCallback, useEffect, useTransition } from "react";
import { useForm, useWatch } from "react-hook-form";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormSelect } from "@/components/form/formSelect";
import type {
    DatabaseGrantJobResponse,
    DatabaseListResponse,
    DatabaseUserListResponse,
} from "@/contracts/types";
import { api } from "@/lib/api";
import { useDatabaseAction } from "./useDatabaseAction";

type CreateGrantProps = {
    databases: DatabaseListResponse["items"];
    users: DatabaseUserListResponse["items"];
};

const formSchema = v.object({
    databaseId: v.pipe(v.string(), v.nonEmpty("Choose a database.")),
    userId: v.pipe(v.string(), v.nonEmpty("Choose a database user.")),
});

type FormValues = v.InferOutput<typeof formSchema>;

const CreateGrant = ({ databases, users }: CreateGrantProps) => {
    const { run } = useDatabaseAction();
    const [pending, startTransition] = useTransition();
    const form = useForm<FormValues>({
        resolver: valibotResolver(formSchema),
        defaultValues: {
            databaseId: databases[0]?.id ?? "",
            userId: users[0]?.id ?? "",
        },
    });
    const databaseId = useWatch({ control: form.control, name: "databaseId" });
    const userId = useWatch({ control: form.control, name: "userId" });

    useEffect(() => {
        if (!databases.some((item) => item.id === form.getValues("databaseId"))) {
            form.setValue("databaseId", databases[0]?.id ?? "");
        }
        if (!users.some((item) => item.id === form.getValues("userId"))) {
            form.setValue("userId", users[0]?.id ?? "");
        }
    }, [databases, form, users]);

    const handleSubmit = useCallback(
        (values: FormValues) => {
            startTransition(async () => {
                const path = `databases/${encodeURIComponent(values.databaseId)}/users/${encodeURIComponent(values.userId)}`;
                await run(
                    () => api.put(path).json<DatabaseGrantJobResponse>(),
                    "Access granted",
                );
            });
        },
        [run],
    );

    return (
        <Form {...form}>
            <form className="panel-card p-6" onSubmit={form.handleSubmit(handleSubmit)}>
                <h2 className="text-base font-semibold">Add grant</h2>
                <FormSelect
                    className="mt-5"
                    name="databaseId"
                    label="Database"
                    options={databases}
                    required
                />
                <FormSelect
                    className="mt-4"
                    name="userId"
                    label="User"
                    options={users}
                    required
                />
                <Button
                    className="mt-5"
                    type="submit"
                    variant="primary"
                    fullWidth
                    isDisabled={!databaseId || !userId || pending}
                >
                    Grant access
                </Button>
            </form>
        </Form>
    );
};

export default CreateGrant;
