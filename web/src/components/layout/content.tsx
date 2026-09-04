import type { ComponentProps } from "react";
import { cn } from "@/utils/classnames";

type ContentProps = ComponentProps<"main">;

export const Content = ({ className, ...props }: ContentProps) => (
    <main
        className={cn(
            "mx-auto w-full max-w-[100rem] px-5 py-7 sm:px-8 sm:py-9 lg:px-10 lg:py-10",
            className,
        )}
        {...props}
    />
);
