"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { Button, Spinner, toast } from "@heroui/react";
import { Save } from "lucide-react";
import { useRouter } from "next/navigation";
import { useCallback, useTransition } from "react";
import { useForm } from "react-hook-form";
import { useSWRConfig } from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormInput } from "@/components/form/formInput";
import { FormSelect } from "@/components/form/formSelect";
import type { AuthResponse, UpdateProfileRequest } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isValidTimezone, TIMEZONES } from "@/utils/timezones";
import {
    emailField,
    nameField,
    passwordField,
    usernameField,
} from "@/utils/validation";

type Props = {
    forced?: boolean;
    session: AuthResponse;
};

const newPassword = v.pipe(passwordField, v.minLength(12, "Use at least 12 characters."));
const timezoneField = v.pipe(
    v.string(),
    v.check((value) => isValidTimezone(value), "Choose a valid timezone."),
);
const timezoneOptions = TIMEZONES.map(({ label, value }) => ({ id: value, name: label }));

const getSchema = (forced: boolean) =>
    v.pipe(
        v.object({
            username: usernameField,
            name: nameField,
            email: emailField,
            timezone: timezoneField,
            currentPassword: v.pipe(
                v.string(),
                v.maxLength(128, "Password must be 128 characters or fewer."),
            ),
            password: forced ? newPassword : v.union([v.literal(""), newPassword]),
            confirmation: v.string(),
        }),
        v.forward(
            v.partialCheck(
                [["password"], ["confirmation"]],
                ({ password, confirmation }) => password === confirmation,
                "New password confirmation does not match.",
            ),
            ["confirmation"],
        ),
    );

type FormValues = v.InferOutput<ReturnType<typeof getSchema>>;

export const ProfileForm = ({ forced = false, session }: Props) => {
    const router = useRouter();
    const { mutate } = useSWRConfig();
    const [pending, startTransition] = useTransition();
    const form = useForm<FormValues>({
        resolver: valibotResolver(getSchema(forced)),
        defaultValues: {
            username: session.user.username,
            name: session.user.name,
            email: forced ? "" : session.user.email,
            timezone: session.timezone,
            currentPassword: "",
            password: "",
            confirmation: "",
        },
    });

    const handleSubmit = useCallback(
        (values: FormValues) => {
            const data: UpdateProfileRequest = {
                username: values.username,
                name: values.name,
                email: values.email,
                timezone: values.timezone,
            };
            if (values.currentPassword) data.currentPassword = values.currentPassword;
            if (values.password) data.password = values.password;

            startTransition(async () => {
                try {
                    await api.patch("auth/profile", { json: data });
                    form.resetField("currentPassword");
                    form.resetField("password");
                    form.resetField("confirmation");
                    await mutate("auth/me");
                    router.refresh();
                    toast.success(forced ? "Administrator configured" : "Profile updated");
                } catch (error) {
                    toast.danger("Profile update failed", {
                        description: await errorMessage(error),
                    });
                }
            });
        },
        [forced, form, mutate, router],
    );

    return (
        <Form {...form}>
            <form className="space-y-5" onSubmit={form.handleSubmit(handleSubmit)}>
                <div className="grid gap-5 sm:grid-cols-2">
                    <FormInput
                        name="username"
                        label="Username"
                        autoComplete="username"
                        required
                    />
                    <FormInput name="name" label="Full name" autoComplete="name" required />
                </div>

                <FormInput
                    name="email"
                    label="Email address"
                    type="email"
                    autoComplete="email"
                    required
                />

                <FormSelect
                    name="timezone"
                    label="Timezone"
                    options={timezoneOptions}
                    required
                />

                {!forced && (
                    <FormInput
                        name="currentPassword"
                        label="Current password"
                        type="password"
                        autoComplete="current-password"
                    />
                )}

                <div className="grid gap-5 sm:grid-cols-2">
                    <FormInput
                        name="password"
                        label={forced ? "New password" : "New password (optional)"}
                        type="password"
                        autoComplete="new-password"
                        required={forced}
                    />
                    <FormInput
                        name="confirmation"
                        label="Confirm new password"
                        type="password"
                        autoComplete="new-password"
                        required={forced}
                    />
                </div>

                <div className="flex justify-end pt-2">
                    <Button type="submit" variant="primary" isPending={pending}>
                        {pending ? (
                            <Spinner color="current" size="sm" />
                        ) : (
                            <Save className="size-4" />
                        )}
                        {forced ? "Complete setup" : "Save changes"}
                    </Button>
                </div>
            </form>
        </Form>
    );
};
