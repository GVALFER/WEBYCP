"use client";

import { AlertDialog, Button, Spinner } from "@heroui/react";
import { TriangleAlert } from "lucide-react";
import { type ReactNode, useState } from "react";

type Props = {
    children: ReactNode;
    title: string;
    description: string;
    action?: string;
    onConfirm: () => Promise<unknown> | void;
};

export const Confirm = ({ children, title, description, action = "Confirm", onConfirm }: Props) => {
    const [pending, setPending] = useState(false);

    const confirm = async (close: () => void) => {
        setPending(true);

        try {
            await onConfirm();
            close();
        } finally {
            setPending(false);
        }
    };

    return (
        <AlertDialog>
            <AlertDialog.Trigger>{children}</AlertDialog.Trigger>
            <AlertDialog.Backdrop>
                <AlertDialog.Container placement="center" size="sm">
                    <AlertDialog.Dialog>
                        {({ close }) => (
                            <>
                                <AlertDialog.Header>
                                    <AlertDialog.Icon status="danger">
                                        <TriangleAlert aria-hidden="true" />
                                    </AlertDialog.Icon>
                                    <AlertDialog.Heading>{title}</AlertDialog.Heading>
                                </AlertDialog.Header>
                                <AlertDialog.Body>{description}</AlertDialog.Body>
                                <AlertDialog.Footer>
                                    <Button
                                        variant="tertiary"
                                        isDisabled={pending}
                                        onPress={close}
                                    >
                                        Cancel
                                    </Button>
                                    <Button
                                        variant="danger"
                                        isPending={pending}
                                        onPress={() => void confirm(close)}
                                    >
                                        {pending ? (
                                            <Spinner color="current" size="sm" />
                                        ) : null}
                                        {action}
                                    </Button>
                                </AlertDialog.Footer>
                            </>
                        )}
                    </AlertDialog.Dialog>
                </AlertDialog.Container>
            </AlertDialog.Backdrop>
        </AlertDialog>
    );
};
