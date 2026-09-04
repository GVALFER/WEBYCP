"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { Button, Modal, Spinner } from "@heroui/react";
import { useCallback } from "react";
import { useForm } from "react-hook-form";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormInput } from "@/components/form/formInput";

type Props = {
    open: boolean;
    title: string;
    label: string;
    value: string;
    schema: v.GenericSchema<string>;
    pending?: boolean;
    onOpenChange: (open: boolean) => void;
    onSubmit: (value: string) => Promise<void> | void;
};

type FormValues = { value: string };

export const TextDialog = ({ open, ...props }: Props) =>
    open ? <TextDialogContent {...props} /> : null;

const TextDialogContent = ({
    title,
    label,
    value,
    schema,
    pending = false,
    onOpenChange,
    onSubmit,
}: Omit<Props, "open">) => {
    const formSchema = v.object({ value: schema });
    const form = useForm<FormValues>({
        resolver: valibotResolver(formSchema),
        defaultValues: { value },
    });
    const handleSubmit = useCallback(
        async (values: FormValues) => onSubmit(values.value),
        [onSubmit],
    );

    return (
        <Modal isOpen onOpenChange={onOpenChange}>
            <Modal.Backdrop>
                <Modal.Container placement="center" size="sm">
                    <Modal.Dialog>
                        <Form {...form}>
                            <form onSubmit={form.handleSubmit(handleSubmit)}>
                                <Modal.Header>
                                    <Modal.Heading>{title}</Modal.Heading>
                                </Modal.Header>
                                <Modal.Body>
                                    <FormInput
                                        name="value"
                                        label={label}
                                        autoFocus
                                        required
                                    />
                                </Modal.Body>
                                <Modal.Footer>
                                    <Button
                                        variant="tertiary"
                                        isDisabled={pending}
                                        onPress={() => onOpenChange(false)}
                                    >
                                        Cancel
                                    </Button>
                                    <Button type="submit" variant="primary" isPending={pending}>
                                        {pending ? (
                                            <Spinner color="current" size="sm" />
                                        ) : null}
                                        Save changes
                                    </Button>
                                </Modal.Footer>
                            </form>
                        </Form>
                    </Modal.Dialog>
                </Modal.Container>
            </Modal.Backdrop>
        </Modal>
    );
};
