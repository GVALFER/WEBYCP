"use client";

import * as v from "valibot";
import { FormInput } from "@/components/form/formInput";
import type { Package, WritePackageRequest } from "@/contracts/types";
import { nameField } from "@/utils/validation";

const count = v.pipe(
    v.number("Enter a limit."),
    v.integer("Use a whole number."),
    v.minValue(0, "Use 0 or more."),
    v.maxValue(1_000_000, "Use 1000000 or fewer."),
);

export const packageSchema = v.pipe(
    v.object({
        name: nameField,
        limits: v.object({
            websites: count,
            domains: count,
            aliases: count,
            databases: count,
            databaseUsers: count,
            scheduledTasks: count,
            backupPlans: count,
            backupRetention: v.pipe(
                v.number("Enter a retention."),
                v.integer("Use a whole number."),
                v.minValue(1, "Keep at least one backup."),
                v.maxValue(100, "Keep 100 backups or fewer."),
            ),
        }),
    }),
    v.forward(
        v.check(
            (value) => value.limits.domains >= value.limits.websites,
            "Domains must be equal to or greater than Websites.",
        ),
        ["limits", "domains"],
    ),
);

export type PackageValues = v.InferOutput<typeof packageSchema>;

export const packageValues = (value?: Package): WritePackageRequest => ({
    name: value?.name ?? "",
    limits: value?.limits ?? {
        websites: 10,
        domains: 10,
        aliases: 20,
        databases: 10,
        databaseUsers: 10,
        scheduledTasks: 20,
        backupPlans: 5,
        backupRetention: 7,
    },
});

export const PackageFields = () => (
    <div className="grid gap-4 sm:grid-cols-2">
        <FormInput className="sm:col-span-2" name="name" label="Package name" required />
        <FormInput name="limits.websites" label="Websites" type="number" min={0} required />
        <FormInput name="limits.domains" label="Domains" type="number" min={0} required />
        <FormInput name="limits.aliases" label="Aliases" type="number" min={0} required />
        <FormInput name="limits.databases" label="Databases" type="number" min={0} required />
        <FormInput
            name="limits.databaseUsers"
            label="Database users"
            type="number"
            min={0}
            required
        />
        <FormInput
            name="limits.scheduledTasks"
            label="Scheduled tasks"
            type="number"
            min={0}
            required
        />
        <FormInput name="limits.backupPlans" label="Backup plans" type="number" min={0} required />
        <FormInput
            name="limits.backupRetention"
            label="Backup retention"
            type="number"
            min={1}
            max={100}
            required
        />
    </div>
);
