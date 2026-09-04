import { cn } from "@/utils/classnames";
import { Card, type Card as CardTypes } from "@heroui/react";

type NextCardProps = {
    children: React.ReactNode | string | undefined;
    header?: React.ReactNode | string | undefined;
    footer?: React.ReactNode | string | undefined;
    className?: string;
} & CardTypes["Props"];

export const NextCard = ({ children, header, footer, className, ...props }: NextCardProps) => {
    return (
        <Card className={cn("w-full p-3 sm:p-5", className)} {...props}>
            {header && <Card.Header>{header}</Card.Header>}
            {children && <Card.Content>{children}</Card.Content>}
            {footer && <Card.Footer>{footer}</Card.Footer>}
        </Card>
    );
};
