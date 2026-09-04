"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { Button, toast } from "@heroui/react";
import { LockKeyhole } from "lucide-react";
import { useRouter } from "next/navigation";
import { useCallback, useTransition } from "react";
import { useForm } from "react-hook-form";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormInput } from "@/components/form/formInput";
import { api } from "@/lib/api";
import { getReturnTo } from "@/lib/status";
import { errorMessage } from "@/utils/errors";
import { passwordField, usernameField } from "@/utils/validation";

const formSchema = v.object({
    username: usernameField,
    password: passwordField,
});

type FormValues = v.InferOutput<typeof formSchema>;

const LoginForm = () => {
    const router = useRouter();
    const [pending, startTransition] = useTransition();
    const form = useForm<FormValues>({
        resolver: valibotResolver(formSchema),
        defaultValues: {
            username: "",
            password: "",
        },
    });

    const handleSubmit = useCallback(
        (values: FormValues) => {
            startTransition(async () => {
                try {
                    await api.post("auth/login", { json: values });
                    const returnTo = getReturnTo(
                        new URLSearchParams(window.location.search).get("returnTo"),
                    );
                    router.replace(returnTo);
                    router.refresh();
                } catch (error) {
                    toast.danger("Sign in failed", {
                        description: await errorMessage(error),
                    });
                }
            });
        },
        [router],
    );

    return (
        <Form {...form}>
            <form className="space-y-5" onSubmit={form.handleSubmit(handleSubmit)}>
                <FormInput
                    name="username"
                    label="Username"
                    autoComplete="username"
                    required
                />
                <FormInput
                    name="password"
                    label="Password"
                    type="password"
                    autoComplete="current-password"
                    required
                />
                <Button
                    className="mt-2"
                    type="submit"
                    variant="primary"
                    fullWidth
                    isDisabled={pending}
                >
                    <LockKeyhole className="size-4" aria-hidden="true" />
                    {pending ? "Please wait…" : "Sign in"}
                </Button>
            </form>
        </Form>
    );
};

export default LoginForm;
