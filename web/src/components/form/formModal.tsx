"use client";

import { Button, Modal, Spinner } from "@heroui/react";
import { Plus } from "lucide-react";
import type { ComponentProps, ReactNode } from "react";

type Props = {
    children: ReactNode;
    description: string;
    open: boolean;
    pending?: boolean;
    size?: ComponentProps<typeof Modal.Container>["size"];
    submitDisabled?: boolean;
    submitLabel: string;
    title: string;
    triggerLabel: string;
    triggerIcon?: ReactNode;
    triggerText?: string;
    triggerVariant?: ComponentProps<typeof Button>["variant"];
    onOpenChange: (open: boolean) => void;
    onSubmit: ComponentProps<"form">["onSubmit"];
};

export const FormModal = ({
    children,
    description,
    open,
    pending = false,
    size = "md",
    submitDisabled = false,
    submitLabel,
    title,
    triggerLabel,
    triggerIcon,
    triggerText,
    triggerVariant = "primary",
    onOpenChange,
    onSubmit,
}: Props) => (
    <>
        <Button
            isIconOnly={!triggerText}
            size="sm"
            variant={triggerVariant}
            aria-label={triggerLabel}
            onPress={() => onOpenChange(true)}
        >
            {triggerIcon ?? <Plus className="size-4" aria-hidden="true" />}
            {triggerText}
        </Button>

        {open && (
            <Modal isOpen onOpenChange={onOpenChange}>
                <Modal.Backdrop variant="blur">
                    <Modal.Container placement="center" size={size}>
                        <Modal.Dialog>
                            <form onSubmit={onSubmit}>
                                <Modal.CloseTrigger aria-label="Close" />
                                <Modal.Header>
                                    <Modal.Heading>{title}</Modal.Heading>
                                    <div className="mt-1 text-sm font-normal text-foreground-500">
                                        {description}
                                    </div>
                                </Modal.Header>
                                <Modal.Body className="space-y-4">{children}</Modal.Body>
                                <Modal.Footer>
                                    <Button
                                        variant="tertiary"
                                        isDisabled={pending}
                                        onPress={() => onOpenChange(false)}
                                    >
                                        Cancel
                                    </Button>
                                    <Button
                                        type="submit"
                                        variant="primary"
                                        isPending={pending}
                                        isDisabled={submitDisabled}
                                    >
                                        {pending ? <Spinner color="current" size="sm" /> : null}
                                        {submitLabel}
                                    </Button>
                                </Modal.Footer>
                            </form>
                        </Modal.Dialog>
                    </Modal.Container>
                </Modal.Backdrop>
            </Modal>
        )}
    </>
);
