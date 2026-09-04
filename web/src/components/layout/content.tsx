import type { ComponentProps } from "react";
import { cn } from "@/utils/classnames";

type ContentProps = ComponentProps<"main">;

export const Content = ({ className, ...props }: ContentProps) => (
    <main
        className={cn(
            "mx-auto w-full max-w-[100rem] px-5 py-6 sm:px-8 sm:py-8 lg:px-10",
            className,
        )}
        {...props}
    />
);
